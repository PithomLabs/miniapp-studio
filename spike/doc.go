// Package spike holds the P0 framework validation spike: it proves the Studio
// module imports the exact pinned Miniapp revision and drives one Class A
// invocation plus one approval park/decide/same-key-resumption round-trip
// through the canonical runtime path.
//
// It requires TEST_DATABASE_URL pointing at a scratch PostgreSQL 19 database
// (the suite truncates framework tables). Studio CI provides it; without the
// variable the tests skip following the framework's own test convention.
package spike
