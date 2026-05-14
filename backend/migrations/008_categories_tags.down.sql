-- Rollback for migration 008.
DELETE FROM role_permissions WHERE permission_id IN (
    SELECT id FROM permissions WHERE (resource, action) IN (
        ('categories', 'manage'),
        ('tags',       'read'),
        ('tags',       'manage')
    )
);

DELETE FROM permissions WHERE (resource, action) IN (
    ('categories', 'manage'),
    ('tags',       'read'),
    ('tags',       'manage')
);

DROP TABLE IF EXISTS tags;
DROP TABLE IF EXISTS categories;
