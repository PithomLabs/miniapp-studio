package store

import (
	"context"
	"fmt"
	"strings"

	"github.com/PithomLabs/miniapp-studio/dbenv"
	"github.com/jackc/pgx/v5/pgxpool"
)

// dumpFunctionACLs renders every studio/app function with its owner and
// full grantee list (grantee=privilege/grantor), plus default ACLs and
// the runtime roles' superuser flags. A privilege mismatch fails WITH
// its mechanism instead of a bare boolean. Secret-free: ACLs name
// roles, never credentials.
func dumpFunctionACLs(ctx context.Context, owner *pgxpool.Pool) string {
	var b strings.Builder
	rows, err := owner.Query(ctx, `
SELECT n.nspname || '.' || p.proname || '(' ||
       pg_get_function_identity_arguments(p.oid) || ')' AS fn,
       o.rolname AS fowner,
       COALESCE((SELECT string_agg(g.rolname || '=' || x.privilege_type ||
                                   '/' || gr.rolname, ',' ORDER BY 1)
                 FROM aclexplode(p.proacl) x
                 JOIN pg_roles g ON g.oid = x.grantee
                 JOIN pg_roles gr ON gr.oid = x.grantor),
                'NULL=defaults(owner+X,PUBLIC+X)') AS acl
FROM pg_proc p
JOIN pg_namespace n ON n.oid = p.pronamespace
JOIN pg_roles o ON o.oid = p.proowner
WHERE n.nspname IN ('studio', 'app')
ORDER BY 1`)
	if err != nil {
		return fmt.Sprintf("acl dump query: %v", err)
	}
	for rows.Next() {
		var fn, fowner, acl string
		if err := rows.Scan(&fn, &fowner, &acl); err != nil {
			b.WriteString(fmt.Sprintf("acl scan: %v\n", err))
			break
		}
		fmt.Fprintf(&b, "fn %s owner=%s acl=[%s]\n", fn, fowner, acl)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		fmt.Fprintf(&b, "acl rows: %v\n", err)
	}
	rows, err = owner.Query(ctx, `
SELECT COALESCE(d.defaclnamespace::regnamespace::text, '(global)') AS scope,
       d.defaclobjtype AS objtype,
       r.rolname AS for_role,
       COALESCE((SELECT string_agg(g.rolname || '=' || x.privilege_type, ',')
                 FROM aclexplode(d.defaclacl) x
                 JOIN pg_roles g ON g.oid = x.grantee), '(none)') AS acl
FROM pg_default_acl d
JOIN pg_roles r ON r.oid = d.defaclrole`)
	if err != nil {
		fmt.Fprintf(&b, "default-acl dump query: %v\n", err)
	} else {
		empty := true
		for rows.Next() {
			empty = false
			var scope, objtype, forRole, acl string
			if err := rows.Scan(&scope, &objtype, &forRole, &acl); err != nil {
				fmt.Fprintf(&b, "default-acl scan: %v\n", err)
				break
			}
			fmt.Fprintf(&b, "default_acl scope=%s obj=%s for=%s acl=[%s]\n",
				scope, objtype, forRole, acl)
		}
		rows.Close()
		if empty {
			b.WriteString("default_acl: (none)\n")
		}
	}
	rows, err = owner.Query(ctx, `
SELECT rolname, rolsuper, rolbypassrls, rolcanlogin
FROM pg_roles WHERE rolname IN ('studio_app', 'studio_lookup', $1)
ORDER BY 1`, dbenv.TestRoleName)
	if err != nil {
		fmt.Fprintf(&b, "role dump query: %v\n", err)
		return b.String()
	}
	for rows.Next() {
		var name string
		var super, bypass, login bool
		if err := rows.Scan(&name, &super, &bypass, &login); err != nil {
			fmt.Fprintf(&b, "role scan: %v\n", err)
			break
		}
		fmt.Fprintf(&b, "role %s super=%v bypassrls=%v login=%v\n",
			name, super, bypass, login)
	}
	rows.Close()
	return b.String()
}

// dumpRoleDeps lists everything tying a scratch role to the database:
// owned objects, shared dependencies, memberships either direction,
// and held grants. Included in a DROP ROLE failure so the 2BP01 names
// its dependent instead of guessing.
func dumpRoleDeps(ctx context.Context, owner *pgxpool.Pool, role string) string {
	rows, err := owner.Query(ctx, `
SELECT 'owns relation ' || n.nspname || '.' || c.relname || ' (' || c.relkind || ')'
FROM pg_class c
JOIN pg_namespace n ON n.oid = c.relnamespace
JOIN pg_roles r ON r.oid = c.relowner WHERE r.rolname = $1
UNION ALL
SELECT 'owns function ' || n.nspname || '.' || p.proname || '(' ||
       pg_get_function_identity_arguments(p.oid) || ')'
FROM pg_proc p
JOIN pg_namespace n ON n.oid = p.pronamespace
JOIN pg_roles r ON r.oid = p.proowner WHERE r.rolname = $1
UNION ALL
SELECT 'owns schema ' || n.nspname
FROM pg_namespace n
JOIN pg_roles r ON r.oid = n.nspowner WHERE r.rolname = $1
UNION ALL
SELECT 'shdepend ' || d.deptype || ' on ' || d.classid::regclass ||
       ' obj ' || d.objid || '/' || d.objsubid
FROM pg_shdepend d
JOIN pg_roles r ON r.oid = d.refobjid
WHERE r.rolname = $1 AND d.refclassid = 'pg_authid'::regclass
UNION ALL
SELECT 'member: ' || m.rolname || ' IN ' || g.rolname ||
       ' admin=' || a.admin_option || ' set=' || a.set_option ||
       ' inherit=' || a.inherit_option
FROM pg_auth_members a
JOIN pg_roles m ON m.oid = a.member
JOIN pg_roles g ON g.oid = a.roleid
WHERE m.rolname = $1 OR g.rolname = $1
UNION ALL
SELECT 'schema grant: ' || n.nspname || ' ' || x.privilege_type
FROM pg_namespace n, aclexplode(n.nspacl) x
JOIN pg_roles r ON r.oid = x.grantee WHERE r.rolname = $1
UNION ALL
SELECT 'function grant: ' || n.nspname || '.' || p.proname || ' ' || x.privilege_type
FROM pg_proc p
JOIN pg_namespace n ON n.oid = p.pronamespace,
aclexplode(p.proacl) x
JOIN pg_roles r ON r.oid = x.grantee WHERE r.rolname = $1
ORDER BY 1`, role)
	if err != nil {
		return fmt.Sprintf("dependency dump query: %v", err)
	}
	defer rows.Close()
	var b strings.Builder
	empty := true
	for rows.Next() {
		empty = false
		var line string
		if err := rows.Scan(&line); err != nil {
			fmt.Fprintf(&b, "dependency scan: %v\n", err)
			break
		}
		b.WriteString(line + "\n")
	}
	if empty {
		b.WriteString("(no owned objects, dependencies, memberships, or grants found)\n")
	}
	return b.String()
}
