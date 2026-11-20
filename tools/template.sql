--
-- PostgreSQL database dump
--

\restrict bIuRJNjEQiBJZzShggSU3KsrNAbqZCqseTYrXwTVMMzR1Mt7aWmUPp5ICEvuZYk

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
-- Name: consonant; Type: TABLE; Schema: public; Owner: goclient
--

CREATE TABLE public.consonant (
    piece integer,
    "offset" integer,
    duration integer,
    pitches character varying[],
    midi_values integer[],
    note_ids integer[],
    voices integer[],
    pitches_dissonant boolean,
    midi_dissonant boolean
);


ALTER TABLE public.consonant OWNER TO goclient;

--
-- Name: dissonance; Type: TABLE; Schema: public; Owner: goclient
--

CREATE TABLE public.dissonance (
    piece integer,
    "offset" integer,
    duration integer,
    pitches character varying[],
    midi_values integer[],
    note_ids integer[],
    voices integer[],
    pitches_dissonant boolean,
    midi_dissonant boolean
);


ALTER TABLE public.dissonance OWNER TO goclient;

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
-- Name: sonority; Type: TABLE; Schema: public; Owner: goclient
--

CREATE TABLE public.sonority (
    piece integer,
    "offset" integer,
    duration integer,
    pitches character varying[],
    midi_values integer[],
    note_ids integer[],
    voices integer[]
);


ALTER TABLE public.sonority OWNER TO goclient;

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
-- Name: TABLE staves; Type: ACL; Schema: public; Owner: absum
--

GRANT SELECT,INSERT,UPDATE ON TABLE public.staves TO goclient;


--
-- Name: TABLE voces; Type: ACL; Schema: public; Owner: absum
--

GRANT SELECT ON TABLE public.voces TO goclient;


--
-- PostgreSQL database dump complete
--

\unrestrict bIuRJNjEQiBJZzShggSU3KsrNAbqZCqseTYrXwTVMMzR1Mt7aWmUPp5ICEvuZYk

