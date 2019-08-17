CREATE SCHEMA saaras_db;
CREATE TABLE saaras_db.org (
    org_id bigint NOT NULL,
    org_name character varying NOT NULL,
    create_ts timestamp with time zone DEFAULT now(),
    update_ts timestamp with time zone DEFAULT now(),
    delete_flag boolean DEFAULT false
);
COMMENT ON TABLE saaras_db.org IS 'Table to hold org data';
CREATE SEQUENCE saaras_db.org_org_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER SEQUENCE saaras_db.org_org_id_seq OWNED BY saaras_db.org.org_id;
CREATE TABLE saaras_db.proxy (
    proxy_id bigint NOT NULL,
    proxy_name character varying NOT NULL,
    create_ts timestamp with time zone DEFAULT now() NOT NULL,
    update_ts timestamp with time zone DEFAULT now() NOT NULL,
    delete_flag boolean DEFAULT false
);
CREATE SEQUENCE saaras_db.proxy_proxy_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER SEQUENCE saaras_db.proxy_proxy_id_seq OWNED BY saaras_db.proxy.proxy_id;
CREATE TABLE saaras_db.proxy_service (
    proxy_id bigint NOT NULL,
    service_id bigint NOT NULL,
    proxy_service_id bigint NOT NULL,
    create_ts timestamp with time zone NOT NULL,
    update_ts timestamp with time zone NOT NULL,
    delete_flag boolean
);
CREATE SEQUENCE saaras_db.proxy_service_proxy_service_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER SEQUENCE saaras_db.proxy_service_proxy_service_id_seq OWNED BY saaras_db.proxy_service.proxy_service_id;
CREATE TABLE saaras_db.route (
    route_id bigint NOT NULL,
    route_name character varying NOT NULL,
    prefix text NOT NULL,
    create_ts timestamp with time zone DEFAULT now(),
    update_ts timestamp with time zone DEFAULT now(),
    delete_flag boolean DEFAULT false,
    service_id bigint NOT NULL
);
CREATE SEQUENCE saaras_db.route_route_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER SEQUENCE saaras_db.route_route_id_seq OWNED BY saaras_db.route.route_id;
CREATE TABLE saaras_db.route_temp (
    route_name character varying NOT NULL,
    route_prefix character varying NOT NULL,
    service_name character varying NOT NULL
);
CREATE TABLE saaras_db.route_upstream (
    route_upstream_id bigint NOT NULL,
    route_id bigint NOT NULL,
    upstream_id bigint NOT NULL,
    create_ts timestamp with time zone DEFAULT now(),
    update_ts timestamp with time zone DEFAULT now(),
    delete_flag boolean DEFAULT false
);
CREATE SEQUENCE saaras_db.route_upstream_route_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER SEQUENCE saaras_db.route_upstream_route_id_seq OWNED BY saaras_db.route_upstream.route_id;
CREATE SEQUENCE saaras_db.route_upstream_route_upstream_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER SEQUENCE saaras_db.route_upstream_route_upstream_id_seq OWNED BY saaras_db.route_upstream.route_upstream_id;
CREATE SEQUENCE saaras_db.route_upstream_upstream_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER SEQUENCE saaras_db.route_upstream_upstream_id_seq OWNED BY saaras_db.route_upstream.upstream_id;
CREATE TABLE saaras_db.service (
    service_id bigint NOT NULL,
    service_name character varying NOT NULL,
    fqdn character varying NOT NULL,
    create_ts timestamp with time zone DEFAULT now(),
    update_ts timestamp with time zone DEFAULT now(),
    delete_flag boolean DEFAULT false
);
CREATE SEQUENCE saaras_db.service_service_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER SEQUENCE saaras_db.service_service_id_seq OWNED BY saaras_db.service.service_id;
CREATE TABLE saaras_db.service_temp (
    service_name character varying NOT NULL,
    fqdn character varying NOT NULL
);
CREATE TABLE saaras_db.upstream (
    upstream_id bigint NOT NULL,
    upstream_name character varying NOT NULL,
    upstream_ip character varying NOT NULL,
    upstream_port integer NOT NULL,
    create_ts timestamp with time zone DEFAULT now(),
    update_ts timestamp with time zone DEFAULT now(),
    delete_flag boolean DEFAULT false
);
CREATE SEQUENCE saaras_db.upstream_upstream_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER SEQUENCE saaras_db.upstream_upstream_id_seq OWNED BY saaras_db.upstream.upstream_id;
ALTER TABLE ONLY saaras_db.org ALTER COLUMN org_id SET DEFAULT nextval('saaras_db.org_org_id_seq'::regclass);
ALTER TABLE ONLY saaras_db.proxy ALTER COLUMN proxy_id SET DEFAULT nextval('saaras_db.proxy_proxy_id_seq'::regclass);
ALTER TABLE ONLY saaras_db.proxy_service ALTER COLUMN proxy_service_id SET DEFAULT nextval('saaras_db.proxy_service_proxy_service_id_seq'::regclass);
ALTER TABLE ONLY saaras_db.route ALTER COLUMN route_id SET DEFAULT nextval('saaras_db.route_route_id_seq'::regclass);
ALTER TABLE ONLY saaras_db.route_upstream ALTER COLUMN route_upstream_id SET DEFAULT nextval('saaras_db.route_upstream_route_upstream_id_seq'::regclass);
ALTER TABLE ONLY saaras_db.route_upstream ALTER COLUMN route_id SET DEFAULT nextval('saaras_db.route_upstream_route_id_seq'::regclass);
ALTER TABLE ONLY saaras_db.route_upstream ALTER COLUMN upstream_id SET DEFAULT nextval('saaras_db.route_upstream_upstream_id_seq'::regclass);
ALTER TABLE ONLY saaras_db.service ALTER COLUMN service_id SET DEFAULT nextval('saaras_db.service_service_id_seq'::regclass);
ALTER TABLE ONLY saaras_db.upstream ALTER COLUMN upstream_id SET DEFAULT nextval('saaras_db.upstream_upstream_id_seq'::regclass);
ALTER TABLE ONLY saaras_db.org
    ADD CONSTRAINT org_pkey PRIMARY KEY (org_id);
ALTER TABLE ONLY saaras_db.proxy
    ADD CONSTRAINT proxy_pkey PRIMARY KEY (proxy_id);
ALTER TABLE ONLY saaras_db.proxy_service
    ADD CONSTRAINT proxy_service_pkey PRIMARY KEY (proxy_service_id);
ALTER TABLE ONLY saaras_db.route
    ADD CONSTRAINT route_pkey PRIMARY KEY (route_id);
ALTER TABLE ONLY saaras_db.route_temp
    ADD CONSTRAINT route_temp_pkey PRIMARY KEY (route_name);
ALTER TABLE ONLY saaras_db.route_upstream
    ADD CONSTRAINT route_upstream_pkey PRIMARY KEY (route_upstream_id);
ALTER TABLE ONLY saaras_db.service
    ADD CONSTRAINT service_pkey PRIMARY KEY (service_id);
ALTER TABLE ONLY saaras_db.service
    ADD CONSTRAINT service_service_name_key UNIQUE (service_name);
ALTER TABLE ONLY saaras_db.service_temp
    ADD CONSTRAINT service_temp_pkey PRIMARY KEY (service_name);
ALTER TABLE ONLY saaras_db.upstream
    ADD CONSTRAINT upstream_pkey PRIMARY KEY (upstream_id);
ALTER TABLE ONLY saaras_db.proxy_service
    ADD CONSTRAINT proxy_service_proxy_id_fkey FOREIGN KEY (proxy_id) REFERENCES saaras_db.proxy(proxy_id) ON UPDATE RESTRICT ON DELETE RESTRICT;
ALTER TABLE ONLY saaras_db.proxy_service
    ADD CONSTRAINT proxy_service_service_id_fkey FOREIGN KEY (service_id) REFERENCES saaras_db.service(service_id) ON UPDATE RESTRICT ON DELETE RESTRICT;
ALTER TABLE ONLY saaras_db.route
    ADD CONSTRAINT route_service_id_fkey FOREIGN KEY (service_id) REFERENCES saaras_db.service(service_id) ON UPDATE RESTRICT ON DELETE RESTRICT;
ALTER TABLE ONLY saaras_db.route_upstream
    ADD CONSTRAINT route_upstream_route_id_fkey FOREIGN KEY (route_id) REFERENCES saaras_db.route(route_id) ON UPDATE RESTRICT ON DELETE RESTRICT;
ALTER TABLE ONLY saaras_db.route_upstream
    ADD CONSTRAINT route_upstream_upstream_id_fkey FOREIGN KEY (upstream_id) REFERENCES saaras_db.upstream(upstream_id) ON UPDATE RESTRICT ON DELETE RESTRICT;
