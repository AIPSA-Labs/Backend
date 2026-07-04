-- name: GetOrganizationByUser :one
SELECT o.* FROM organizations o
JOIN organization_members om ON o.id = om.organization_id
WHERE om.user_id = $1
LIMIT 1;

-- name: CreateOrganization :one
INSERT INTO organizations (name, slug, owner_id)
VALUES ($1, $2, $3)
RETURNING *;

-- name: AddOrganizationMember :exec
INSERT INTO organization_members (organization_id, user_id, role)
VALUES ($1, $2, $3);

-- name: IsOrganizationMember :one
SELECT EXISTS(
    SELECT 1 FROM organization_members
    WHERE organization_id = $1 AND user_id = $2
);
