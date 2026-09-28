UPDATE permissions
SET name = 'management.audits.show',
    description = 'can show data audit',
    feature_group = 'management - audits'
WHERE name = 'management.activity-logs.show';
