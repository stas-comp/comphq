// Package people holds the name picker, the person cookie, and the
// person-identity middleware (SPEC B2, B4).
package people

import (
	"database/sql"
	"errors"
	"sort"
	"strings"
	"time"
)

// Person is one row of the people table.
type Person struct {
	ID     int64
	Name   string
	Active bool
}

// ErrDuplicateName is returned by Create when the name (case-insensitively)
// already belongs to an active person (SPEC B3: "unique among active").
var ErrDuplicateName = errors.New("that name is already in use")

// ErrEmptyName is returned by Create for blank or whitespace-only input.
var ErrEmptyName = errors.New("name can't be empty")

// Store reads and writes the people table.
type Store struct {
	DB *sql.DB
}

// List returns every active person, alphabetically (case-insensitive; SPEC
// gate 1.01: "every current name as a button, in alphabetical order").
func (s *Store) List() ([]Person, error) {
	rows, err := s.DB.Query(`SELECT id, name, active FROM people WHERE active = 1`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var people []Person
	for rows.Next() {
		var p Person
		var active int
		if err := rows.Scan(&p.ID, &p.Name, &active); err != nil {
			return nil, err
		}
		p.Active = active != 0
		people = append(people, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	sort.Slice(people, func(i, j int) bool {
		return strings.ToLower(people[i].Name) < strings.ToLower(people[j].Name)
	})
	return people, nil
}

// Get returns the person for id, and whether they exist and are active.
// A missing or inactive person is treated the same by the caller (SPEC
// B4: "the cookie is missing, unknown or inactive").
func (s *Store) Get(id int64) (Person, bool, error) {
	var p Person
	var active int
	err := s.DB.QueryRow(`SELECT id, name, active FROM people WHERE id = ?`, id).
		Scan(&p.ID, &p.Name, &active)
	if err == sql.ErrNoRows {
		return Person{}, false, nil
	}
	if err != nil {
		return Person{}, false, err
	}
	p.Active = active != 0
	return p, p.Active, nil
}

// Create adds a new active person. The name is trimmed; duplicate-checking
// and storage both ignore case.
func (s *Store) Create(name string) (Person, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Person{}, ErrEmptyName
	}

	var exists int
	if err := s.DB.QueryRow(
		`SELECT COUNT(*) FROM people WHERE active = 1 AND LOWER(name) = LOWER(?)`, name,
	).Scan(&exists); err != nil {
		return Person{}, err
	}
	if exists > 0 {
		return Person{}, ErrDuplicateName
	}

	res, err := s.DB.Exec(
		`INSERT INTO people (name, active, created_at) VALUES (?, 1, ?)`,
		name, time.Now().UTC().Format(time.RFC3339),
	)
	if err != nil {
		return Person{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Person{}, err
	}
	return Person{ID: id, Name: name, Active: true}, nil
}

// Deactivate removes a person from the picker and assignment lists; their
// name still shows in existing history (SPEC gate 1.05, 1.06).
func (s *Store) Deactivate(id int64) error {
	_, err := s.DB.Exec(`UPDATE people SET active = 0 WHERE id = ?`, id)
	return err
}
