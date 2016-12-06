CREATE TABLE xmp_yondu_tariffs
(
  id SERIAL PRIMARY KEY NOT NULL,
  value VARCHAR(125) NOT NULL DEFAULT '',
  tariff DOUBLE PRECISION NOT NULL
);
INSERT INTO xmp_yondu_tariffs ( value, tariff ) VALUES
  ('P1',	1),
  ('P250',	2.50),
  ('P5',	5),
  ('10',	10),
  ('15',	15),
  ('20',	20),
  ('25',	25),
  ('30',	30),
  ('35',	35),
  ('40',	40),
  ('45',	45),
  ('50',	50),
  ('55',	55),
  ('60',	60),
  ('65',	65),
  ('70',	70),
  ('75',	75),
  ('80',	80),
  ('85',	85),
  ('90',	95),
  ('100',	100),
  ('150',	150),
  ('200',	200),
  ('250',	250),
  ('300',	300),
  ('350',	350),
  ('400',	400),
  ('450',	450),
  ('500',	500),
  ('550',	550),
  ('600',	600),
  ('650',	650),
  ('700',	700),
  ('750',	750),
  ('800',	800),
  ('850',	850),
  ('900',	900),
  ('950',	950),
  ('1000',	1000);