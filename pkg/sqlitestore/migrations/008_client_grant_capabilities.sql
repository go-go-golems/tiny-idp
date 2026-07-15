-- Existing serialized clients predate explicit grant capabilities. Classify
-- the two shipped public profiles deterministically; deliberately leave every
-- ambiguous legacy client with an empty list so production validation fails
-- closed and an operator must choose its capability set explicitly.
UPDATE clients
SET data = json_set(
    data,
    '$.AllowedGrantTypes',
    CASE
        WHEN json_array_length(json_extract(data, '$.RedirectURIs')) > 0
            THEN json_array('authorization_code', 'refresh_token')
        WHEN json_extract(data, '$.Public') = 1
             AND json_extract(data, '$.RequirePKCE') = 1
            THEN json_array('urn:ietf:params:oauth:grant-type:device_code')
        ELSE json_array()
    END
)
WHERE json_type(data, '$.AllowedGrantTypes') IS NULL
   OR json_type(data, '$.AllowedGrantTypes') = 'null';
