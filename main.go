package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB = nil

func migrate() error {
	db, err := connectDb()
	if err != nil {
		return err
	}
	tables := []string{
		"create table teachers ( id integer primary key, name varchar(200))",
		"create table students ( id integer primary key, name varchar(200));",
		"create table sessions ( id integer primary key, date timestamp);",
		"create table classes ( id integer primary key, name varchar(200), students varchar(600), teacher int);",
	}
	for _, t := range tables {
		fmt.Println("creating tbale ", t)
		_, err = db.Exec(t)
		if err != nil {
			return err
		}
	}
	return nil
}
func connectDb() (*sql.DB, error) {
	if err := db.Ping; err == nil {
		return db, nil
	}
	db, err := sql.Open("sqlite3", "db.sqlite3")
	if err != nil {
		return nil, err
	}
	err = db.Ping()
	if err != nil {
		return nil, err
	}
	return db, nil
}

func dbQuery(query string, args ...interface{}) (*sql.Rows, error) {
	return db.Query(query, args...)
}

func dbExec(stmt string, args ...interface{}) (sql.Result, error) {
	return db.Exec(stmt, args...)
}

func main() {
	// err := migrate()
	// if err != nil {
	// 	panic(err)
	// }
	studentHandler := studentHttpHandler()
	sessionsHandler := sessionHttpHandler()
	classesHandler := classHttpHandler()
	teacherHandler := teacherHttpHandler()
	http.Handle("/students", studentHandler)
	http.Handle("/sessions", sessionsHandler)
	http.Handle("/classes", classesHandler)
	http.Handle("/teachers", teacherHandler)
	err = http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal(err.Error())
	}
}

type Student struct {
	Id   int
	Name string
}

func (s *Student) SqlInsert() string {
	return fmt.Sprintf("INSERT INTO `students` (`id`, `name`) VALUES (%d, %s)", s.Id, s.Name)
}

func (s *Student) SqlUpdate() string {
	return fmt.Sprintf("UPDATE `students` WHERE id=%d SET name=%s", s.Id, s.Name)
}

type Session struct {
	Id   int
	Date time.Time
}

func (s *Session) SqlInsert() string {
	return fmt.Sprintf("INSERT INTO `sessions` (`id`, `date`) VALUES (%d, %d)", s.Id, s.Date.Unix())
}

func (s *Session) SqlUpdate() string {
	return fmt.Sprintf("UPDATE `sessions` WHERE id=%d SET date=%d", s.Id, s.Date.Unix())
}

type Teacher struct {
	Id   int
	Name string
}

func (s *Teacher) SqlInsert() string {
	return fmt.Sprintf("INSERT INTO `teachers` (`id`, `name`) VALUES (%d, %s)", s.Id, s.Name)
}

func (s *Teacher) SqlUpdate() string {
	return fmt.Sprintf("UPDATE `teachers` WHERE id=%d SET name=%s", s.Id, s.Name)
}

type Class struct {
	Id       int
	Name     string
	Students []string
	Teacher  int
}

func (s *Class) studentsCommaSep() string {
	return strings.Join(s.Students, ",")
}
func (s *Class) SqlInsert() string {
	return fmt.Sprintf("INSERT INTO `classes` (`id`, `name`, `students`. `teacher`) VALUES (%d, %s, %s, %d)", s.Id, s.Name, s.studentsCommaSep(), s.Teacher)
}

func (s *Class) SqlUpdate() string {
	return fmt.Sprintf("UPDATE `classes` WHERE id=%d SET name=%s, students=%s, teachers=%d", s.Id, s.Name, s.studentsCommaSep(), s.Teacher)
}

func (c *Class) AddStudent(studentId string) {
	c.Students = append(c.Students, studentId)
}
func (c *Class) RemoveStudent(studentId string) {
	for idx, s := range c.Students {
		if s == studentId {
			c.Students = append(c.Students[:idx], c.Students[idx+1:]...)
			break
		}
	}
}

type Model interface {
	SqlInsert() string
	SqlUpdate() string
}

func get(model interface{}, query string) error {
	rows, err := dbQuery(query)
	if err != nil {
		return err
	}
	for rows.Next() {
		err = rows.Scan(model)
		if err != nil {
			return err
		}
	}
	return nil
}

func Get(modelName string, query string) (interface{}, error) {
	switch modelName {
	case "student":
		m := Student{}
		err := get(&m, query)
		return m, err
	case "session":
		m := Session{}
		err := get(&m, query)
		return m, err
	case "teacher":
		m := Teacher{}
		err := get(&m, query)
		return m, err
	case "class":
		m := Class{}
		err := get(&m, query)
		return m, err
	default:
		return nil, errors.New("No case matched for model")
	}
}

func Save(model Model) (int64, error) {
	result, err := dbExec(model.SqlInsert())
	if err != nil {
		return 0, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	return id, err
}

func Update(model Model) error {
	_, err := db.Exec(model.SqlUpdate())
	return err
}

func sessionHttpHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			if r.URL.Query().Get("id") != "" {
				// get id
				rows, err := dbQuery("SELECT * FROM `sessions` WHERE id=?", r.URL.Query().Get("id"))
				if err != nil {
					w.WriteHeader(500)
					fmt.Fprint(w, err.Error())
					return
				}
				session := &Session{}

				for rows.Next() {
					err = rows.Scan(&session.Id, session.Date)
					if err != nil {
						w.WriteHeader(500)
						fmt.Fprint(w, err.Error())
						return
					}

				}
				bs, err := json.Marshal(session)
				if err != nil {
					w.WriteHeader(500)
					fmt.Fprint(w, err.Error())
					return
				}
				w.Write(bs)
				return
			}
			// get id
			rows, err := dbQuery("SELECT * FROM `sessions`")
			if err != nil {
				w.WriteHeader(500)
				fmt.Fprint(w, err.Error())
				return
			}
			sessions := []*Session{}

			for rows.Next() {
				session := &Session{}
				err = rows.Scan(&session.Id, session.Date)
				if err != nil {
					w.WriteHeader(500)
					fmt.Fprint(w, err.Error())
					return
				}
				sessions = append(sessions, session)
			}
			bs, err := json.Marshal(sessions)
			if err != nil {
				w.WriteHeader(500)
				fmt.Fprint(w, err.Error())
				return
			}
			w.Write(bs)
			return
			//index

		} else if r.Method == http.MethodPost {
			body, err := ioutil.ReadAll(r.Body)
			defer r.Body.Close()
			if err != nil {
				w.WriteHeader(500)
				fmt.Fprint(w, err.Error())
				return
			}
			session := &Session{}
			err = json.Unmarshal(body, session)
			if err != nil {
				w.WriteHeader(500)
				fmt.Fprint(w, err.Error())
				return
			}
			id, err := Save(session)
			if err != nil {
				w.WriteHeader(500)
				fmt.Fprint(w, err.Error())
				return
			}
			w.WriteHeader(201)
			fmt.Fprintf(w, "%d", id)
			return

		} else if r.Method == http.MethodPut {
			body, err := ioutil.ReadAll(r.Body)
			defer r.Body.Close()
			if err != nil {
				w.WriteHeader(500)
				fmt.Fprint(w, err.Error())
				return
			}
			session := &Session{}
			err = json.Unmarshal(body, session)
			if err != nil {
				w.WriteHeader(500)
				fmt.Fprint(w, err.Error())
				return
			}
			err = Update(session)
			if err != nil {
				w.WriteHeader(500)
				fmt.Fprint(w, err.Error())
				return
			}
			w.WriteHeader(200)
			return

		} else if r.Method == http.MethodDelete {
			_, err := dbExec("DELETE FROM `sessions` WHERE id=?", r.URL.Query().Get("id"))
			if err != nil {
				w.WriteHeader(500)
				fmt.Fprint(w, err.Error())
				return
			}
			w.WriteHeader(200)
			return
		} else {
			w.WriteHeader(400)
			fmt.Fprintf(w, "Method %s not supported", r.Method)
			return
		}
	})

	return mux
}
func studentHttpHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			if r.URL.Query().Get("id") != "" {
				// get id
				rows, err := dbQuery("SELECT * FROM `students` WHERE id=?", r.URL.Query().Get("id"))
				if err != nil {
					w.WriteHeader(500)
					fmt.Fprint(w, err.Error())
					return
				}
				student := &Student{}

				for rows.Next() {
					err = rows.Scan(&student.Id, student.Name)
					if err != nil {
						w.WriteHeader(500)
						fmt.Fprint(w, err.Error())
						return
					}

				}
				bs, err := json.Marshal(student)
				if err != nil {
					w.WriteHeader(500)
					fmt.Fprint(w, err.Error())
					return
				}
				w.Write(bs)
				return
			}
			// get id
			rows, err := dbQuery("SELECT * FROM `students`")
			if err != nil {
				w.WriteHeader(500)
				fmt.Fprint(w, err.Error())
				return
			}
			students := []*Student{}

			for rows.Next() {
				student := &Student{}
				err = rows.Scan(&student.Id, student.Name)
				if err != nil {
					w.WriteHeader(500)
					fmt.Fprint(w, err.Error())
					return
				}
				students = append(students, student)
			}
			bs, err := json.Marshal(students)
			if err != nil {
				w.WriteHeader(500)
				fmt.Fprint(w, err.Error())
				return
			}
			w.Write(bs)
			return
			//index

		} else if r.Method == http.MethodPost {
			body, err := ioutil.ReadAll(r.Body)
			defer r.Body.Close()
			if err != nil {
				w.WriteHeader(500)
				fmt.Fprint(w, err.Error())
				return
			}
			student := &Student{}
			err = json.Unmarshal(body, student)
			if err != nil {
				w.WriteHeader(500)
				fmt.Fprint(w, err.Error())
				return
			}
			id, err := Save(student)
			if err != nil {
				w.WriteHeader(500)
				fmt.Fprint(w, err.Error())
				return
			}
			w.WriteHeader(201)
			fmt.Fprintf(w, "%d", id)
			return

		} else if r.Method == http.MethodPut {
			body, err := ioutil.ReadAll(r.Body)
			defer r.Body.Close()
			if err != nil {
				w.WriteHeader(500)
				fmt.Fprint(w, err.Error())
				return
			}
			student := &Student{}
			err = json.Unmarshal(body, student)
			if err != nil {
				w.WriteHeader(500)
				fmt.Fprint(w, err.Error())
				return
			}
			err = Update(student)
			if err != nil {
				w.WriteHeader(500)
				fmt.Fprint(w, err.Error())
				return
			}
			w.WriteHeader(200)
			return

		} else if r.Method == http.MethodDelete {
			_, err := dbExec("DELETE FROM `students` WHERE id=?", r.URL.Query().Get("id"))
			if err != nil {
				w.WriteHeader(500)
				fmt.Fprint(w, err.Error())
				return
			}
			w.WriteHeader(200)
			return
		} else {
			w.WriteHeader(400)
			fmt.Fprintf(w, "Method %s not supported", r.Method)
			return
		}
	})

	return mux
}
func classHttpHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			if r.URL.Query().Get("id") != "" {
				// get id
				rows, err := dbQuery("SELECT * FROM `classes` WHERE id=?", r.URL.Query().Get("id"))
				if err != nil {
					w.WriteHeader(500)
					fmt.Fprint(w, err.Error())
					return
				}
				class := &Class{}

				for rows.Next() {
					students := ""
					err = rows.Scan(&class.Id, &class.Name, &students, class.Teacher)
					class.Students = strings.Split(students, ",")
					if err != nil {
						w.WriteHeader(500)
						fmt.Fprint(w, err.Error())
						return
					}

				}
				bs, err := json.Marshal(class)
				if err != nil {
					w.WriteHeader(500)
					fmt.Fprint(w, err.Error())
					return
				}
				w.Write(bs)
				return
			}
			// get id
			rows, err := dbQuery("SELECT * FROM `classes`")
			if err != nil {
				w.WriteHeader(500)
				fmt.Fprint(w, err.Error())
				return
			}
			classs := []*Class{}

			for rows.Next() {
				class := &Class{}
				students := ""
				err = rows.Scan(&class.Id, &class.Name, &students, class.Teacher)
				class.Students = strings.Split(students, ",")
				if err != nil {
					w.WriteHeader(500)
					fmt.Fprint(w, err.Error())
					return
				}
				classs = append(classs, class)
			}
			bs, err := json.Marshal(classs)
			if err != nil {
				w.WriteHeader(500)
				fmt.Fprint(w, err.Error())
				return
			}
			w.Write(bs)
			return
			//index

		} else if r.Method == http.MethodPost {
			body, err := ioutil.ReadAll(r.Body)
			defer r.Body.Close()
			if err != nil {
				w.WriteHeader(500)
				fmt.Fprint(w, err.Error())
				return
			}
			class := &Class{}
			err = json.Unmarshal(body, class)
			if err != nil {
				w.WriteHeader(500)
				fmt.Fprint(w, err.Error())
				return
			}
			id, err := Save(class)
			if err != nil {
				w.WriteHeader(500)
				fmt.Fprint(w, err.Error())
				return
			}
			w.WriteHeader(201)
			fmt.Fprintf(w, "%d", id)
			return

		} else if r.Method == http.MethodPut {
			body, err := ioutil.ReadAll(r.Body)
			defer r.Body.Close()
			if err != nil {
				w.WriteHeader(500)
				fmt.Fprint(w, err.Error())
				return
			}
			class := &Class{}
			err = json.Unmarshal(body, class)
			if err != nil {
				w.WriteHeader(500)
				fmt.Fprint(w, err.Error())
				return
			}
			err = Update(class)
			if err != nil {
				w.WriteHeader(500)
				fmt.Fprint(w, err.Error())
				return
			}
			w.WriteHeader(200)
			return

		} else if r.Method == http.MethodDelete {
			_, err := dbExec("DELETE FROM `classes` WHERE id=?", r.URL.Query().Get("id"))
			if err != nil {
				w.WriteHeader(500)
				fmt.Fprint(w, err.Error())
				return
			}
			w.WriteHeader(200)
			return
		} else {
			w.WriteHeader(400)
			fmt.Fprintf(w, "Method %s not supported", r.Method)
			return
		}
	})

	return mux
}
func teacherHttpHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			if r.URL.Query().Get("id") != "" {
				// get id
				rows, err := dbQuery("SELECT * FROM `teachers` WHERE id=?", r.URL.Query().Get("id"))
				if err != nil {
					w.WriteHeader(500)
					fmt.Fprint(w, err.Error())
					return
				}
				teacher := &Teacher{}

				for rows.Next() {
					err = rows.Scan(&teacher.Id, teacher.Name)
					if err != nil {
						w.WriteHeader(500)
						fmt.Fprint(w, err.Error())
						return
					}

				}
				bs, err := json.Marshal(teacher)
				if err != nil {
					w.WriteHeader(500)
					fmt.Fprint(w, err.Error())
					return
				}
				w.Write(bs)
				return
			}
			// get id
			rows, err := dbQuery("SELECT * FROM `teachers`")
			if err != nil {
				w.WriteHeader(500)
				fmt.Fprint(w, err.Error())
				return
			}
			teachers := []*Teacher{}

			for rows.Next() {
				teacher := &Teacher{}
				err = rows.Scan(&teacher.Id, teacher.Name)
				if err != nil {
					w.WriteHeader(500)
					fmt.Fprint(w, err.Error())
					return
				}
				teachers = append(teachers, teacher)
			}
			bs, err := json.Marshal(teachers)
			if err != nil {
				w.WriteHeader(500)
				fmt.Fprint(w, err.Error())
				return
			}
			w.Write(bs)
			return
			//index

		} else if r.Method == http.MethodPost {
			body, err := ioutil.ReadAll(r.Body)
			defer r.Body.Close()
			if err != nil {
				w.WriteHeader(500)
				fmt.Fprint(w, err.Error())
				return
			}
			teacher := &Teacher{}
			err = json.Unmarshal(body, teacher)
			if err != nil {
				w.WriteHeader(500)
				fmt.Fprint(w, err.Error())
				return
			}
			id, err := Save(teacher)
			if err != nil {
				w.WriteHeader(500)
				fmt.Fprint(w, err.Error())
				return
			}
			w.WriteHeader(201)
			fmt.Fprintf(w, "%d", id)
			return

		} else if r.Method == http.MethodPut {
			body, err := ioutil.ReadAll(r.Body)
			defer r.Body.Close()
			if err != nil {
				w.WriteHeader(500)
				fmt.Fprint(w, err.Error())
				return
			}
			teacher := &Teacher{}
			err = json.Unmarshal(body, teacher)
			if err != nil {
				w.WriteHeader(500)
				fmt.Fprint(w, err.Error())
				return
			}
			err = Update(teacher)
			if err != nil {
				w.WriteHeader(500)
				fmt.Fprint(w, err.Error())
				return
			}
			w.WriteHeader(200)
			return

		} else if r.Method == http.MethodDelete {
			_, err := dbExec("DELETE FROM `teachers` WHERE id=?", r.URL.Query().Get("id"))
			if err != nil {
				w.WriteHeader(500)
				fmt.Fprint(w, err.Error())
				return
			}
			w.WriteHeader(200)
			return
		} else {
			w.WriteHeader(400)
			fmt.Fprintf(w, "Method %s not supported", r.Method)
			return
		}
	})

	return mux
}
