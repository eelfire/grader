package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	_ "github.com/mattn/go-sqlite3"
	_ "github.com/tursodatabase/libsql-client-go/libsql"
)

type Templates struct {
	templates *template.Template
}

func (t *Templates) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

func NewTemplates() *Templates {
	return &Templates{
		templates: template.Must(template.ParseGlob("views/*.html")),
	}
}

func createDirectory() {
	folderPath := "./db"

	err := os.Mkdir(folderPath, os.ModePerm)
	if err != nil {
		fmt.Println("Error creating folder:", err)
		os.Exit(1)
	}
}

func DB() *sql.DB {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// createDirectory()

	dbName := "LOCAL_DB"
	dbUrl := os.Getenv(dbName)
	if dbUrl == "" {
		fmt.Fprintf(os.Stderr, "missing env %s\n", dbName)
		os.Exit(1)
	}
	db, err := sql.Open("sqlite3", dbUrl)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open db %s: %s", dbUrl, err)
		os.Exit(1)
	}
	createTable(db)
	return db
}

func createTable(db *sql.DB) {
	query := `
		CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			email VARCHAR(255) NOT NULL,
			cpi FLOAT NOT NULL
		)
	`

	_, err := db.Exec(query)
	if err != nil {
		log.Fatalf("failed to create table: %s", err)
	}
}

type Mark struct {
	Id         string
	Name       string
	Score      float32
	MaxScore   float32
	Percentage float32
	Weightage  float32
	Weighted   float32
}

func (m *Mark) updateMarkName(name string) {
	m.Name = name
}

func (m *Mark) updateMarkScore(score float32) {
	m.Score = score
	percentage := float32(score) / float32(m.MaxScore) * 100
	m.Percentage = percentage
	m.Weighted = float32(m.Weightage) * percentage / 100
}

func (m *Mark) updateMarkMaxScore(maxScore float32) {
	m.MaxScore = maxScore
	percentage := float32(m.Score) / float32(maxScore) * 100
	m.Percentage = percentage
	m.Weighted = float32(m.Weightage) * percentage / 100
}

func (m *Mark) updateMarkWeightage(weightage float32) {
	m.Weightage = weightage
	m.Weighted = float32(weightage) * m.Percentage / 100
}

func generateRandomCode(size int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	code := make([]byte, size)
	for i := range code {
		code[i] = charset[rand.Intn(len(charset))]
	}

	return string(code)
}

func (m *Mark) seedMark() {
	m.Id = generateRandomCode(4)
	m.Name = generateRandomCode(6)
	m.Score = rand.Float32() * 100.0         // Generate random score between 0 and 100
	m.MaxScore = m.Score + rand.Float32()*10 // Generate random max score between 0 and 100
	m.Weightage = rand.Float32() * 50        // Generate random weightage between 0 and 50
	m.Percentage = float32(m.Score) / float32(m.MaxScore) * 100
	m.Weighted = float32(m.Weightage) * m.Percentage / 100
}

type Course struct {
	Id             string
	Code           string
	Name           string
	Marks          []Mark
	TotalWeightage float32
	TotalWeighted  float32
}

func (c *Course) calculateTotalWeightage() {
	var totalWeightage float32
	for _, mark := range c.Marks {
		totalWeightage += mark.Weightage
	}
	c.TotalWeightage = totalWeightage
}

func (c *Course) calculateTotalWeighted() {
	var totalWeighted float32
	for _, mark := range c.Marks {
		totalWeighted += mark.Weighted
	}
	c.TotalWeighted = totalWeighted
}

func (c *Course) updateCourseCode(code string) {
	c.Code = code
}

func (c *Course) updateCourseName(name string) {
	c.Name = name
}

func (c *Course) seedCourse() {
	c.Id = generateRandomCode(4)
	c.Code = genRandomCourseCode()
	c.Name = generateRandomCode(6)

	for i := 0; i < 3; i++ {
		mark := Mark{}
		mark.seedMark()
		c.Marks = append(c.Marks, mark)
	}

	c.calculateTotalWeightage()
	c.calculateTotalWeighted()
}

func genRandomCourseCode() string {
	// Generate random uppercase letters
	letter1 := string(rand.Intn(26) + 65)
	letter2 := string(rand.Intn(26) + 65)

	// Generate random digits
	digits := rand.Intn(900) + 100

	return letter1 + letter2 + " " + strconv.Itoa(digits)
}

type Courses struct {
	Courses []Course
}

func (c *Courses) addCourse(course Course) {
	c.Courses = append(c.Courses, course)
}

func (c *Courses) seedCourses() {
	for i := 0; i < 3; i++ {
		course := Course{}
		course.seedCourse()
		c.addCourse(course)
	}
}

func main() {
	db := DB()

	// Insert sample entry into the table
	insertSampleEntry(db)

	e := echo.New()
	e.Renderer = NewTemplates()
	e.Use(middleware.Logger())

	e.Static("/static", "static")

	e.GET("/crow", func(c echo.Context) error {
		return c.String(http.StatusOK, "crow")
	})

	courses := Courses{}
	e.GET("/", func(c echo.Context) error {
		courses.seedCourses()
		return c.Render(http.StatusOK, "index", courses)
	})

	e.POST("/api/courses/add", func(c echo.Context) error {
		newCourse := Course{}
		newCourse.seedCourse()
		courses.addCourse(newCourse)
		return c.Render(http.StatusOK, "course", newCourse)
	})

	e.POST("/api/courses/marks/:id", func(c echo.Context) error {
		id := c.Param("id")
		courseId := strings.Split(id, "/")[0]
		markId := strings.Split(id, "/")[1]

		if markId == "add" {
			newMark := Mark{}
			newMark.seedMark()

			// update courses with this newMark
			for i, course := range courses.Courses {
				if course.Id == courseId {
					courses.Courses[i].Marks = append(courses.Courses[i].Marks, newMark)
				}
			}

			return c.Render(http.StatusOK, "marks", newMark)
		} else if markId == "total" {
			var totalWeightage float32
			var totalWeighted float32
			for _, course := range courses.Courses {
				if course.Id == courseId {
					totalWeightage = course.TotalWeightage
					totalWeighted = course.TotalWeighted
				}
			}
			totalWeightageString := strconv.FormatFloat(float64(totalWeightage), 'f', 2, 32)
			totalWeightedString := strconv.FormatFloat(float64(totalWeighted), 'f', 2, 32)

			return c.HTML(http.StatusOK, "<div>Total Weightage: "+totalWeightageString+"</div><div>Total Weighted: "+totalWeightedString+"</div>")
		} else {
			return c.String(http.StatusOK, "id: "+id)
		}
		// newMarks := Mark{Id: generateRandomCode(4), Name: "kite", Score: 10, MaxScore: 20, Percentage: 50, Weightage: 10, Weighted: 5}
		// return c.Render(http.StatusOK, "marks", newMarks)
	})

	e.PUT("/api/courses/:id", func(c echo.Context) error {
		courseId := c.Param("id")
		newCode := c.FormValue("Code")
		newName := c.FormValue("Name")
		fmt.Println(newCode, newName)

		newCourse := Course{}
		for i, course := range courses.Courses {
			if course.Id == courseId {
				if newCode != "" {
					courses.Courses[i].updateCourseCode(newCode)
				} else if newName != "" {
					courses.Courses[i].updateCourseName(newName)
				}
				newCourse = courses.Courses[i]
			}
		}

		fmt.Println(newCourse)

		// return c.Render(http.StatusOK, "course", newCourse)
		return nil
	})

	e.PUT("/api/courses/marks/:id", func(c echo.Context) error {
		markId := c.Param("id")
		newName := c.FormValue("Name")
		newScore := c.FormValue("Score")
		newMaxScore := c.FormValue("MaxScore")
		newWeightage := c.FormValue("Weightage")
		fmt.Println(newName, newScore, newMaxScore, newWeightage)

		courseId := c.FormValue("CourseId")
		fmt.Println(courseId)

		newMarks := Mark{}
		for i, course := range courses.Courses {
			if course.Id == courseId {
				for j, mark := range course.Marks {
					if mark.Id == markId {
						if newName != "" {
							courses.Courses[i].Marks[j].updateMarkName(newName)
						} else if newScore != "" {
							newScoreInt := stringToFloat(newScore)
							courses.Courses[i].Marks[j].updateMarkScore(newScoreInt)
						} else if newMaxScore != "" {
							newMaxScoreInt := stringToFloat(newMaxScore)
							courses.Courses[i].Marks[j].updateMarkMaxScore(newMaxScoreInt)
						} else if newWeightage != "" {
							newWeightageInt := stringToFloat(newWeightage)
							courses.Courses[i].Marks[j].updateMarkWeightage(newWeightageInt)
						}
						newMarks = courses.Courses[i].Marks[j]

						courses.Courses[i].calculateTotalWeightage()
						courses.Courses[i].calculateTotalWeighted()
					}
				}
			}
		}

		fmt.Println(newMarks)

		return c.Render(http.StatusOK, "marks", newMarks)
	})

	e.Logger.Fatal(e.Start(":7878"))
}

func insertSampleEntry(db *sql.DB) {
	query := `
		INSERT INTO users (email, cpi)
		VALUES ('student@study.com', 7.8)
		ON CONFLICT DO NOTHING
	`

	_, err := db.Exec(query)
	if err != nil {
		log.Fatalf("failed to insert sample entry: %s", err)
	}
}

func stringToFloat(s string) float32 {
	f, err := strconv.ParseFloat(s, 32)
	if err != nil {
		log.Fatalf("failed to convert string to float: %s", err)
	}
	return float32(f)
}
