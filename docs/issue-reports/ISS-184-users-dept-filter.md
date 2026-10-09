# ISS-184 - GET /users department filter semantics

Decision: document, do not align. Users list/detail/mutations stay exact-department; reports keep department + descendants (deptscope).

Why: all user mutations and GET /users/{id} use exact-department (inCallerScope). Widening only the list would expose users the caller cannot open or manage, and widening everything touches the same service.go/repository.go/types.go/scoping_test.go that in-flight #240 (swarm/240-users-authz-d4) rewrites, risking conflicts and a security-sensitive change without a product decision. Security rule holds: department_admin never sees users outside its own department; super_admin unaffected.

RequireUserInScope: already removed from code (see ISS-232 report); Store.UserInScope is used by reports. Only a package doc note was added in deptscope/scope.go (no collision with #240, which does not touch deptscope).

Changes: new backend/internal/users/list_scope_iss184_test.go (3 tests pinning semantics, no edit to #240-touched files), FR-BB117 decision D-5, deptscope doc comment. No migrations, no frontend.
