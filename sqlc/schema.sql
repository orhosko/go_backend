CREATE TABLE team (
    id          INTEGER     PRIMARY KEY,
    name        text        NOT NULL,
    strength    INTEGER
);

CREATE TABLE standing (
    id          INTEGER     PRIMARY KEY,
    points      integer,
    wins        integer,
    draws       integer,
    losses      integer,
    goal_diff   integer
);

CREATE TABLE teamStanding (
    id          INTEGER     PRIMARY KEY,
    team        team,
    standing    standing
);

CREATE TABLE season (
    id          INTEGER     PRIMARY KEY,
    name        text        NOT NULL,
    standings   standing
);

CREATE TABLE match (
    id          INTEGER     PRIMARY KEY,
    home        team,
    guest       team,
    played      bool,
    week        integer
);

CREATE TABLE match_result (
    id          INTEGER     PRIMARY KEY,
    match       match,
    winner      team
);

CREATE TABLE teamStats (
    id          INTEGER     PRIMARY KEY,
    team        team,
    value       integer,
    lastSeasonStanding integer
);

INSERT INTO team (
    name,
    strength
) VALUES (
    'Manchester City',
    10
);

INSERT INTO team (
    name,
    strength
) VALUES (
    'Manchester United',
    8
);

INSERT INTO team (
    name,
    strength
) VALUES (
    'Chelsea',
    7
);

INSERT INTO team (
    name,
    strength
) VALUES (
    'Arsenal',
    6
);

INSERT INTO team (
    name,
    strength
) VALUES(
    "Liverpool",
    9
);
