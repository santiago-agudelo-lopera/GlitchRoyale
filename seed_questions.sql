WITH inserted_question AS (
  INSERT INTO questions (question, category, difficulty)
  VALUES (
    'Cual es el resultado de 2 + 2?',
    'matematicas',
    'easy'
  )
  RETURNING id
)
INSERT INTO answers (question_id, text, is_correct)
SELECT inserted_question.id, answer.text, answer.is_correct
FROM inserted_question
CROSS JOIN (
  VALUES
    ('3', false),
    ('4', true),
    ('5', false),
    ('6', false)
) AS answer(text, is_correct);

WITH inserted_question AS (
  INSERT INTO questions (question, category, difficulty)
  VALUES (
    'Que lenguaje se usa principalmente para dar estilo a una pagina web?',
    'programacion',
    'easy'
  )
  RETURNING id
)
INSERT INTO answers (question_id, text, is_correct)
SELECT inserted_question.id, answer.text, answer.is_correct
FROM inserted_question
CROSS JOIN (
  VALUES
    ('HTML', false),
    ('CSS', true),
    ('SQL', false),
    ('Go', false)
) AS answer(text, is_correct);

WITH inserted_question AS (
  INSERT INTO questions (question, category, difficulty)
  VALUES (
    'En PostgreSQL, que clausula se usa para ordenar resultados?',
    'base de datos',
    'medium'
  )
  RETURNING id
)
INSERT INTO answers (question_id, text, is_correct)
SELECT inserted_question.id, answer.text, answer.is_correct
FROM inserted_question
CROSS JOIN (
  VALUES
    ('GROUP BY', false),
    ('ORDER BY', true),
    ('HAVING', false),
    ('RETURNING', false)
) AS answer(text, is_correct);

WITH inserted_question AS (
  INSERT INTO questions (question, category, difficulty)
  VALUES (
    'Que hook de React se usa normalmente para manejar estado local?',
    'react',
    'medium'
  )
  RETURNING id
)
INSERT INTO answers (question_id, text, is_correct)
SELECT inserted_question.id, answer.text, answer.is_correct
FROM inserted_question
CROSS JOIN (
  VALUES
    ('useEffect', false),
    ('useRef', false),
    ('useState', true),
    ('useContext', false)
) AS answer(text, is_correct);

WITH inserted_question AS (
  INSERT INTO questions (question, category, difficulty)
  VALUES (
    'Que protocolo usa normalmente un navegador para una conexion WebSocket segura?',
    'redes',
    'hard'
  )
  RETURNING id
)
INSERT INTO answers (question_id, text, is_correct)
SELECT inserted_question.id, answer.text, answer.is_correct
FROM inserted_question
CROSS JOIN (
  VALUES
    ('ws://', false),
    ('wss://', true),
    ('http://', false),
    ('tcp://', false)
) AS answer(text, is_correct);

WITH inserted_question AS (
  INSERT INTO questions (question, category, difficulty)
  VALUES (
    'Que significa CRUD en desarrollo de software?',
    'programacion',
    'hard'
  )
  RETURNING id
)
INSERT INTO answers (question_id, text, is_correct)
SELECT inserted_question.id, answer.text, answer.is_correct
FROM inserted_question
CROSS JOIN (
  VALUES
    ('Create, Read, Update, Delete', true),
    ('Copy, Run, Upload, Deploy', false),
    ('Create, Render, Use, Debug', false),
    ('Code, Review, Update, Document', false)
) AS answer(text, is_correct);
