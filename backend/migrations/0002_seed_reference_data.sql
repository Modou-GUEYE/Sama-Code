-- Données de référence : rôles et catégories de permis (CDC §5.1, §6.3).

INSERT INTO roles (code, name) VALUES
    ('candidate', 'Candidat'),
    ('editor',    'Éditeur pédagogique'),
    ('validator', 'Validateur'),
    ('admin',     'Administrateur');

INSERT INTO license_categories
    (code, name, series_count, questions_per_series, max_faults, exam_duration_minutes, sort_order)
VALUES
    ('B', 'Permis B', 12, 25, 4, 20, 1),
    ('C', 'Permis C',  5, 10, 4, 20, 2);
