--
-- PostgreSQL database dump
--

\restrict nwKDONafMKcnixss3YznOnbhckWRhEkiw6z8O5KEHVK6MosLh3S44p3axPk9RMa

-- Dumped from database version 18.6
-- Dumped by pg_dump version 18.6

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET transaction_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: disstable; Type: TABLE; Schema: public; Owner: goclient
--

CREATE TABLE public.disstable (
    piece integer,
    "offset" integer,
    duration integer,
    pitches character varying[],
    midi_values integer[],
    note_ids integer[],
    voices integer[],
    pitches_dissonant boolean,
    midi_dissonant boolean,
    bar integer
);


ALTER TABLE public.disstable OWNER TO goclient;

--
-- Name: intervala; Type: TABLE; Schema: public; Owner: goclient
--

CREATE TABLE public.intervala (
    "row" integer NOT NULL,
    "-4" integer[],
    "-3" integer[],
    "-2" integer[],
    "-1" integer[],
    "0" integer[],
    "1" integer[],
    "2" integer[],
    "3" integer[],
    "4" integer[]
);


ALTER TABLE public.intervala OWNER TO goclient;

--
-- Name: notes; Type: TABLE; Schema: public; Owner: goclient
--

CREATE TABLE public.notes (
    piece integer,
    mvt integer,
    staffchange integer,
    voice integer,
    index integer,
    pitch character varying(2),
    duration integer,
    midi integer,
    inflec integer,
    ficta integer,
    colour boolean,
    bar integer,
    islast integer,
    toffset integer,
    dimlevel integer,
    id integer NOT NULL
);


ALTER TABLE public.notes OWNER TO goclient;

--
-- Name: notes_id_seq; Type: SEQUENCE; Schema: public; Owner: goclient
--

ALTER TABLE public.notes ALTER COLUMN id ADD GENERATED ALWAYS AS IDENTITY (
    SEQUENCE NAME public.notes_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: notes_r1; Type: TABLE; Schema: public; Owner: goclient
--

CREATE TABLE public.notes_r1 (
    piece integer,
    mvt integer,
    voice integer,
    pitch character varying(4),
    duration integer,
    toffset integer,
    id integer NOT NULL
);


ALTER TABLE public.notes_r1 OWNER TO goclient;

--
-- Name: notes_r1_id_seq; Type: SEQUENCE; Schema: public; Owner: goclient
--

ALTER TABLE public.notes_r1 ALTER COLUMN id ADD GENERATED ALWAYS AS IDENTITY (
    SEQUENCE NAME public.notes_r1_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: pieces; Type: TABLE; Schema: public; Owner: absum
--

CREATE TABLE public.pieces (
    seq integer NOT NULL,
    title character varying(100) NOT NULL,
    composer character varying(100) NOT NULL,
    voices integer
);


ALTER TABLE public.pieces OWNER TO absum;

--
-- Name: pieces_seq_seq; Type: SEQUENCE; Schema: public; Owner: absum
--

ALTER TABLE public.pieces ALTER COLUMN seq ADD GENERATED ALWAYS AS IDENTITY (
    SEQUENCE NAME public.pieces_seq_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: sonor; Type: TABLE; Schema: public; Owner: absum
--

CREATE TABLE public.sonor (
    key integer CONSTRAINT sonor_table_key_not_null NOT NULL,
    value integer
);


ALTER TABLE public.sonor OWNER TO absum;

--
-- Name: sonority_view; Type: VIEW; Schema: public; Owner: goclient
--

CREATE VIEW public.sonority_view AS
 WITH change_points AS (
         SELECT notes.piece,
            notes.toffset AS point
           FROM public.notes
        UNION
         SELECT notes.piece,
            (notes.toffset + notes.duration) AS point
           FROM public.notes
        ), intervals AS (
         SELECT change_points.piece,
            change_points.point AS startoffset,
            lead(change_points.point) OVER (PARTITION BY change_points.piece ORDER BY change_points.point) AS endoffset
           FROM change_points
        ), sonority_notes AS (
         SELECT i.piece,
            i.startoffset,
            i.endoffset,
            (i.endoffset - i.startoffset) AS duration,
            n.id AS note_id,
            n.voice,
            n.pitch,
            n.midi,
            n.toffset
           FROM (intervals i
             JOIN public.notes n ON (((n.piece = i.piece) AND (n.toffset <= i.startoffset) AND ((n.toffset + n.duration) >= i.endoffset))))
          WHERE (i.startoffset < i.endoffset)
        ), elementary_sonorities AS (
         SELECT sonority_notes.piece,
            sonority_notes.startoffset,
            sonority_notes.endoffset,
            sonority_notes.duration,
            max(sonority_notes.toffset) AS newest_note_offset,
            array_agg(sonority_notes.pitch ORDER BY sonority_notes.voice) AS pitches,
            array_agg(sonority_notes.midi ORDER BY sonority_notes.voice) AS midi_values,
            array_agg(sonority_notes.note_id ORDER BY sonority_notes.voice) AS note_ids,
            array_agg(sonority_notes.voice ORDER BY sonority_notes.voice) AS voices
           FROM sonority_notes
          GROUP BY sonority_notes.piece, sonority_notes.startoffset, sonority_notes.endoffset, sonority_notes.duration
        ), islands AS (
         SELECT elementary_sonorities.piece,
            elementary_sonorities.startoffset,
            elementary_sonorities.endoffset,
            elementary_sonorities.duration,
            elementary_sonorities.newest_note_offset,
            elementary_sonorities.pitches,
            elementary_sonorities.midi_values,
            elementary_sonorities.note_ids,
            elementary_sonorities.voices,
                CASE
                    WHEN (lag(elementary_sonorities.pitches) OVER (PARTITION BY elementary_sonorities.piece ORDER BY elementary_sonorities.startoffset) IS DISTINCT FROM elementary_sonorities.pitches) THEN 1
                    ELSE 0
                END AS is_new_sonority
           FROM elementary_sonorities
        ), groups AS (
         SELECT islands.piece,
            islands.startoffset,
            islands.endoffset,
            islands.duration,
            islands.newest_note_offset,
            islands.pitches,
            islands.midi_values,
            islands.note_ids,
            islands.voices,
            islands.is_new_sonority,
            sum(islands.is_new_sonority) OVER (PARTITION BY islands.piece ORDER BY islands.startoffset) AS group_id
           FROM islands
        ), merged_groups AS (
         SELECT groups.piece,
            groups.group_id,
            min(groups.startoffset) AS "offset",
            (max(groups.endoffset) - min(groups.startoffset)) AS duration,
            groups.pitches,
            groups.midi_values,
            groups.voices
           FROM groups
          GROUP BY groups.piece, groups.group_id, groups.pitches, groups.midi_values, groups.voices
        ), merged_note_ids AS (
         SELECT g.piece,
            g.group_id,
            array_agg(DISTINCT u.note_id ORDER BY u.note_id) AS note_ids
           FROM groups g,
            LATERAL unnest(g.note_ids) u(note_id)
          GROUP BY g.piece, g.group_id
        )
 SELECT mg.piece,
    mg."offset",
    mg.duration,
    mg.pitches,
    mg.midi_values,
    mn.note_ids,
    mg.voices
   FROM (merged_groups mg
     JOIN merged_note_ids mn ON (((mg.piece = mn.piece) AND (mg.group_id = mn.group_id))))
  ORDER BY mg.piece, mg."offset";


ALTER VIEW public.sonority_view OWNER TO goclient;

--
-- Name: voces; Type: TABLE; Schema: public; Owner: absum
--

CREATE TABLE public.voces (
    pitch character varying(2) CONSTRAINT voces_table_pitch_not_null NOT NULL,
    "C2" integer,
    "D2" integer,
    "E2" integer,
    "F2" integer,
    "G2" integer,
    "A2" integer,
    "B2" integer,
    "C3" integer,
    "D3" integer,
    "E3" integer,
    "F3" integer,
    "G3" integer,
    "A3" integer,
    "B3" integer,
    "C4" integer,
    "D4" integer,
    "E4" integer,
    "F4" integer,
    "G4" integer,
    "A4" integer,
    "B4" integer,
    "C5" integer,
    "D5" integer,
    "E5" integer,
    "F5" integer,
    "G5" integer,
    "A5" integer,
    "B5" integer,
    "C6" integer,
    "D6" integer
);


ALTER TABLE public.voces OWNER TO absum;

--
-- Name: sonority_dissonance_view; Type: VIEW; Schema: public; Owner: goclient
--

CREATE VIEW public.sonority_dissonance_view AS
 SELECT s.piece,
    s."offset",
    s.duration,
    s.pitches,
    s.midi_values,
    s.note_ids,
    s.voices,
    flags.pitches_dissonant,
    flags.midi_dissonant
   FROM (public.sonority_view s
     CROSS JOIN LATERAL ( WITH sonority_notes AS (
                 SELECT n.pitch,
                    n.midi,
                    n.ordinality
                   FROM UNNEST(s.pitches, s.midi_values) WITH ORDINALITY n(pitch, midi, ordinality)
                  WHERE ((n.pitch IS NOT NULL) AND ((n.pitch)::text <> ''::text) AND ((n.pitch)::text <> '0'::text) AND (n.midi IS NOT NULL) AND (n.midi <> 0))
                ), pair_values AS (
                 SELECT a.pitch AS pitch_a,
                    b.pitch AS pitch_b,
                    LEAST(a.midi, b.midi) AS lower_midi,
                        CASE
                            WHEN (a.midi < b.midi) THEN a.pitch
                            ELSE b.pitch
                        END AS lower_pitch,
                    (abs((a.midi - b.midi)) % 12) AS semitone_interval,
                    ((to_jsonb(va.*) ->> (b.pitch)::text))::integer AS interval_value
                   FROM ((sonority_notes a
                     JOIN sonority_notes b ON ((a.ordinality < b.ordinality)))
                     JOIN public.voces va ON (((va.pitch)::text = (a.pitch)::text)))
                ), pair_support AS (
                 SELECT p.pitch_a,
                    p.pitch_b,
                    p.lower_midi,
                    p.lower_pitch,
                    p.semitone_interval,
                    p.interval_value,
                    COALESCE((EXISTS ( SELECT 1
                           FROM (sonority_notes support_note
                             JOIN public.voces lower_v ON (((lower_v.pitch)::text = (p.lower_pitch)::text)))
                          WHERE ((support_note.midi < p.lower_midi) AND ((((to_jsonb(lower_v.*) ->> (support_note.pitch)::text))::integer % '-7'::integer) = ANY (ARRAY['-2'::integer, '-4'::integer]))))), false) AS has_lower_voces_support,
                    COALESCE((NOT (EXISTS ( SELECT 1
                           FROM sonority_notes support_note
                          WHERE ((support_note.midi < p.lower_midi) AND (((p.lower_midi - support_note.midi) % 12) = ANY (ARRAY[3, 4, 7])))))), true) AS has_no_midi_support
                   FROM pair_values p
                ), pair_flags AS (
                 SELECT p.pitch_a,
                    p.pitch_b,
                    p.lower_midi,
                    p.lower_pitch,
                    p.semitone_interval,
                    p.interval_value,
                    p.has_lower_voces_support,
                    p.has_no_midi_support,
                    (((abs(p.interval_value) % 7) = ANY (ARRAY[1, 6])) OR (((abs(p.interval_value) % 7) = 3) AND (NOT p.has_lower_voces_support))) AS is_voces_dissonant
                   FROM pair_support p
                )
         SELECT COALESCE(bool_or(pair_flags.is_voces_dissonant), false) AS pitches_dissonant,
            COALESCE(bool_or((pair_flags.is_voces_dissonant AND ((pair_flags.semitone_interval = ANY (ARRAY[1, 2, 6, 10, 11])) OR ((pair_flags.semitone_interval = 5) AND pair_flags.has_no_midi_support)))), false) AS midi_dissonant
           FROM pair_flags) flags);


ALTER VIEW public.sonority_dissonance_view OWNER TO goclient;

--
-- Name: staves; Type: TABLE; Schema: public; Owner: absum
--

CREATE TABLE public.staves (
    seq integer NOT NULL,
    num integer,
    denom integer,
    modus integer,
    tempus integer,
    prolatio integer,
    dimin boolean,
    propd integer,
    propn integer,
    modusct integer,
    key integer,
    clef character varying(2)
);


ALTER TABLE public.staves OWNER TO absum;

--
-- Name: staves_seq_seq; Type: SEQUENCE; Schema: public; Owner: absum
--

ALTER TABLE public.staves ALTER COLUMN seq ADD GENERATED ALWAYS AS IDENTITY (
    SEQUENCE NAME public.staves_seq_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: intervala intervala_pkey; Type: CONSTRAINT; Schema: public; Owner: goclient
--

ALTER TABLE ONLY public.intervala
    ADD CONSTRAINT intervala_pkey PRIMARY KEY ("row");


--
-- Name: notes notes_pkey; Type: CONSTRAINT; Schema: public; Owner: goclient
--

ALTER TABLE ONLY public.notes
    ADD CONSTRAINT notes_pkey PRIMARY KEY (id);


--
-- Name: notes_r1 notes_r1_pkey; Type: CONSTRAINT; Schema: public; Owner: goclient
--

ALTER TABLE ONLY public.notes_r1
    ADD CONSTRAINT notes_r1_pkey PRIMARY KEY (id);


--
-- Name: pieces pieces_pkey; Type: CONSTRAINT; Schema: public; Owner: absum
--

ALTER TABLE ONLY public.pieces
    ADD CONSTRAINT pieces_pkey PRIMARY KEY (seq);


--
-- Name: sonor sonor_table_pkey; Type: CONSTRAINT; Schema: public; Owner: absum
--

ALTER TABLE ONLY public.sonor
    ADD CONSTRAINT sonor_table_pkey PRIMARY KEY (key);


--
-- Name: staves staves_pkey; Type: CONSTRAINT; Schema: public; Owner: absum
--

ALTER TABLE ONLY public.staves
    ADD CONSTRAINT staves_pkey PRIMARY KEY (seq);


--
-- Name: voces voces_table_pkey; Type: CONSTRAINT; Schema: public; Owner: absum
--

ALTER TABLE ONLY public.voces
    ADD CONSTRAINT voces_table_pkey PRIMARY KEY (pitch);


--
-- Name: notes notes_piece_fkey; Type: FK CONSTRAINT; Schema: public; Owner: goclient
--

ALTER TABLE ONLY public.notes
    ADD CONSTRAINT notes_piece_fkey FOREIGN KEY (piece) REFERENCES public.pieces(seq);


--
-- Name: notes notes_staffchange_fkey; Type: FK CONSTRAINT; Schema: public; Owner: goclient
--

ALTER TABLE ONLY public.notes
    ADD CONSTRAINT notes_staffchange_fkey FOREIGN KEY (staffchange) REFERENCES public.staves(seq);


--
-- Name: SCHEMA public; Type: ACL; Schema: -; Owner: pg_database_owner
--

GRANT ALL ON SCHEMA public TO goclient;


--
-- Name: TABLE pieces; Type: ACL; Schema: public; Owner: absum
--

GRANT SELECT,INSERT,UPDATE ON TABLE public.pieces TO goclient;


--
-- Name: TABLE sonor; Type: ACL; Schema: public; Owner: absum
--

GRANT SELECT ON TABLE public.sonor TO goclient;


--
-- Name: TABLE voces; Type: ACL; Schema: public; Owner: absum
--

GRANT SELECT ON TABLE public.voces TO goclient;


--
-- Name: TABLE staves; Type: ACL; Schema: public; Owner: absum
--

GRANT SELECT,INSERT,UPDATE ON TABLE public.staves TO goclient;


--
-- PostgreSQL database dump complete
--

\unrestrict nwKDONafMKcnixss3YznOnbhckWRhEkiw6z8O5KEHVK6MosLh3S44p3axPk9RMa

