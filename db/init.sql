CREATE TABLE IF NOT EXISTS `User` (
  Name VARCHAR(255) PRIMARY KEY,
  Score INT
);

INSERT INTO `User` (Name, Score)
VALUES
  ('Tom', 630),
  ('Jack', 589),
  ('Sam', 567),
  ('Alice', 412),
  ('Bob', 731),
  ('Eve', 298),
  ('Mike', 845),
  ('Lily', 512),
  ('Rose', 476),
  ('David', 663),
  ('Jenny', 704),
  ('Leo', 355)
ON DUPLICATE KEY UPDATE Score=VALUES(Score);
