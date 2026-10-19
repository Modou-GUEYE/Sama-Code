-- Acceptation des conditions générales d'utilisation (Q13) : version acceptée et date.
-- NULL = pas encore acceptées (comptes créés avant cette migration) : l'application
-- les fait accepter à la prochaine ouverture.
ALTER TABLE users
    ADD COLUMN terms_version     text,
    ADD COLUMN terms_accepted_at timestamptz;
