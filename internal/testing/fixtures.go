package testing

// SampleGoProject returns a sample Go project structure for testing
func SampleGoProject() map[string]string {
	return map[string]string{
		"go.mod": `module github.com/example/testproject

go 1.22

require (
	github.com/gorilla/mux v1.8.0
	github.com/lib/pq v1.10.0
)
`,
		"main.go": `package main

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
)

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/", HomeHandler)
	r.HandleFunc("/api/users", UsersHandler)

	fmt.Println("Server starting on :8080")
	http.ListenAndServe(":8080", r)
}

func HomeHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Welcome"))
}

func UsersHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Users API"))
}
`,
		"internal/database/db.go": `package database

import (
	"database/sql"
	_ "github.com/lib/pq"
)

type Database struct {
	conn *sql.DB
}

func NewDatabase(connStr string) (*Database, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}
	return &Database{conn: db}, nil
}

func (d *Database) Close() error {
	return d.conn.Close()
}
`,
		"internal/models/user.go": `package models

type User struct {
	ID       int64
	Username string
	Email    string
	Active   bool
}

func NewUser(username, email string) *User {
	return &User{
		Username: username,
		Email:    email,
		Active:   true,
	}
}
`,
		"README.md": `# Test Project

A sample Go web application for testing.

## Features

- REST API with Gorilla Mux
- PostgreSQL database integration
- User management

## Installation

` + "```bash" + `
go build -o app .
./app
` + "```" + `
`,
	}
}

// SamplePythonProject returns a sample Python project structure for testing
func SamplePythonProject() map[string]string {
	return map[string]string{
		"requirements.txt": `flask==2.3.0
sqlalchemy==2.0.0
pydantic==2.0.0
`,
		"app.py": `from flask import Flask, jsonify
from database import db
from models import User

app = Flask(__name__)
app.config['SQLALCHEMY_DATABASE_URI'] = 'sqlite:///app.db'
db.init_app(app)

@app.route('/')
def home():
    return jsonify({"message": "Welcome"})

@app.route('/api/users')
def users():
    users = User.query.all()
    return jsonify([u.to_dict() for u in users])

if __name__ == '__main__':
    app.run(debug=True)
`,
		"database.py": `from flask_sqlalchemy import SQLAlchemy

db = SQLAlchemy()
`,
		"models.py": `from database import db

class User(db.Model):
    id = db.Column(db.Integer, primary_key=True)
    username = db.Column(db.String(80), unique=True, nullable=False)
    email = db.Column(db.String(120), unique=True, nullable=False)
    active = db.Column(db.Boolean, default=True)

    def to_dict(self):
        return {
            'id': self.id,
            'username': self.username,
            'email': self.email,
            'active': self.active
        }
`,
	}
}

// SampleAnalysisOutput returns mock analysis output for testing
func SampleAnalysisOutput() string {
	return `# Code Structure Analysis

## Architectural Overview

This is a Go web application using the Gorilla Mux router framework and PostgreSQL database.

## Core Components

### HTTP Server (main.go)
- Entry point of the application
- Configures routes using Gorilla Mux
- Exposes REST API endpoints

### Database Layer (internal/database/)
- PostgreSQL connection management
- Database abstraction

### Models (internal/models/)
- User entity definition
- Data structure representations

## Service Definitions

The application follows a simple layered architecture:
- **Presentation Layer**: HTTP handlers in main.go
- **Data Layer**: Database package for persistence
- **Domain Layer**: Models package for business entities

## Interface Contracts

### Database Interface
The Database struct provides methods for:
- Connection initialization
- Connection cleanup

### Model Interfaces
User model provides:
- User creation
- User data representation

## Design Patterns Identified

- **Repository Pattern**: Database package abstracts data access
- **Constructor Pattern**: NewDatabase and NewUser factory functions
- **Singleton Pattern**: Database connection is managed centrally

## Component Relationships

main.go → database → PostgreSQL
main.go → models → User entities
HTTP Router → Handlers → Models

## Key Methods & Functions

- main(): Application entry point, server initialization
- HomeHandler(): Handles root endpoint
- UsersHandler(): Handles user API endpoint
- NewDatabase(): Database connection factory
- NewUser(): User entity factory
`
}

// SampleREADME returns a mock README for testing
func SampleREADME() string {
	return `# Test Project

Generated documentation for test project.

## Overview

This is a test project for validating gendocs functionality.

## Architecture

The project follows a standard layered architecture with:
- API layer
- Business logic layer
- Data access layer

## Getting Started

` + "```bash" + `
# Install dependencies
go mod download

# Run the application
go run main.go
` + "```" + `

## API Endpoints

### GET /
Returns welcome message

### GET /api/users
Returns list of users

## Database

PostgreSQL database with the following tables:
- users

## Contributing

Pull requests are welcome.

## License

MIT
`
}

// SampleLLMToolCallResponse returns a mock LLM response with tool calls
func SampleLLMToolCallResponse(toolName string, args map[string]interface{}) string {
	return `I need to examine the files in the repository.`
}
