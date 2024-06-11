package main

import (
	"context"
	"database/sql"
	"errors"
	_ "github.com/mattn/go-sqlite3"
	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	pb "proftests/grpc"
	"strconv"
	"time"
)

type server struct {
	db *sql.DB
	c  pb.ProfileServiceClient
}

func (s *server) makeTest(id int) (test Test) {
	rows, err := s.db.Query("SELECT * from tests WHERE id = ($1)", id)
	if err != nil {
		log.Fatal(err)
	}
	test = Test{}
	test.IsCompleted = false
	for rows.Next() {
		rows.Scan(&test.Id, &test.Label)
	}
	rowsq, err := s.db.Query("SELECT * from questions WHERE test_id = ($1)", test.Id)
	if err != nil {
		log.Fatal(err)
	}
	for rowsq.Next() {
		var question Question
		rowsq.Scan(&question.Id, &question.Test_id, &question.Question)
		if question.Test_id == test.Id {
			test.Questions = append(test.Questions, question)
		}
	}
	rowsa, _ := s.db.Query("SELECT * FROM answers")
	for rowsa.Next() {
		var answer Answer
		rowsa.Scan(&answer.Id, &answer.Question_id, &answer.Answer, &answer.Answer_value, &answer.Result_id)
		for i, question := range test.Questions {
			if question.Id == answer.Question_id {
				test.Questions[i].Answers = append(test.Questions[i].Answers, answer)
			}
		}
	}
	return test
}
func (s *server) Test1(w http.ResponseWriter, r *http.Request) {
	test := s.makeTest(1)
	t, err := template.ParseFiles("templates/test.html")
	if err != nil {
		log.Fatal(err)
	}
	if err := t.Execute(w, test); err != nil {
		log.Fatal(err)
	}
}
func (s *server) Test2(w http.ResponseWriter, r *http.Request) {
	test := s.makeTest(2)
	t, err := template.ParseFiles("templates/test.html")
	if err != nil {
		log.Fatal(err)
	}
	if err = t.Execute(w, test); err != nil {
		log.Fatal(err)
	}
}
func (s *server) Test3(w http.ResponseWriter, r *http.Request) {
	test := s.makeTest(3)
	t, err := template.ParseFiles("templates/test.html")
	if err != nil {
		log.Fatal(err)
	}
	if err = t.Execute(w, test); err != nil {
		log.Fatal(err)
	}
}
func (s *server) SubmitTest1(w http.ResponseWriter, r *http.Request) {
	hasUser := false
	hasResults := false
	test := s.makeTest(1)
	token, _ := s.getToken(r)
	results := s.getResultsPattern(test.Id)
	var userResult Result
	userResult.User_id = 0
	var newUser User
	var user User
	user.Token = ""
	rows, err := s.db.Query("SELECT * FROM users WHERE token = ($1)",
		token,
	)
	if err != nil {
		log.Fatal(err)
	}
	if rows != nil {
		for rows.Next() {
			rows.Scan(&user.Id, &user.Token)
		}
	}
	if user.Token != "" {
		hasUser = true
	}
	if !hasUser {
		_, err = s.db.Exec("INSERT INTO users(token) VALUES ($1)", token)
		if err != nil {
			log.Fatal(err)
		}
		rows1, err1 := s.db.Query("SELECT * FROM users WHERE token = ($1)", token)
		if err1 != nil {
			log.Fatal(err1)
		}
		if rows1 != nil {
			for rows1.Next() {
				rows1.Scan(&newUser.Id, &newUser.Token)
			}
		}
		for _, result := range results {
			sum_value := 0
			for i, question := range test.Questions {
				question.Chosen_answer.Id, _ = strconv.Atoi(r.PostFormValue(strconv.Itoa(question.Id)))
				for _, answer := range question.Answers {
					if answer.Id == question.Chosen_answer.Id {
						question.Chosen_answer = answer
					}
				}
				if result.Id == question.Chosen_answer.Result_id {
					sum_value += question.Chosen_answer.Answer_value
				}
				test.Questions[i] = question
			}
			_, err = s.db.Exec("INSERT INTO results(test_id, result_type, result_value, user_id) VALUES ($1, $2, $3, $4)", test.Id, result.Result_type, sum_value, newUser.Id)

		}
	}
	if hasUser {
		rows, err = s.db.Query("SELECT * FROM results WHERE user_id = ($1) AND test_id = ($2)", user.Id, test.Id)
		if err != nil {
			log.Fatal(err)
		}
		if rows != nil {
			for rows.Next() {
				rows.Scan(&userResult.Id, &userResult.Test_id, &userResult.Result_type, &userResult.Result_value, &userResult.User_id)
			}
		}
		if userResult.Test_id != 0 {
			hasResults = true
		}
		if !hasResults {
			for _, result := range results {
				sum_value := 0
				for i, question := range test.Questions {
					question.Chosen_answer.Id, _ = strconv.Atoi(r.PostFormValue(strconv.Itoa(question.Id)))
					for _, answer := range question.Answers {
						if answer.Id == question.Chosen_answer.Id {
							question.Chosen_answer = answer
						}
					}
					if result.Id == question.Chosen_answer.Result_id {
						sum_value += question.Chosen_answer.Answer_value
					}
					test.Questions[i] = question
				}
				_, err = s.db.Exec("INSERT INTO results(test_id, result_type, result_value, user_id) VALUES ($1, $2, $3, $4)", test.Id, result.Result_type, sum_value, user.Id)
				if err != nil {
					log.Fatal(err)
				}
			}
		}
	}
	if hasUser || hasResults {
		for _, result := range results {
			sum_value := 0
			for i, question := range test.Questions {
				question.Chosen_answer.Id, _ = strconv.Atoi(r.PostFormValue(strconv.Itoa(question.Id)))
				for _, answer := range question.Answers {
					if answer.Id == question.Chosen_answer.Id {
						question.Chosen_answer = answer
					}
				}
				if result.Id == question.Chosen_answer.Result_id {
					sum_value += question.Chosen_answer.Answer_value
				}
				test.Questions[i] = question
			}
			_, err = s.db.Exec("UPDATE results SET result_value = ($1) WHERE test_id = ($2) AND result_type = ($3) AND user_id = ($4)", sum_value, test.Id, result.Result_type, user.Id)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
	results = s.getResults(test.Id, token)
	http.Redirect(w, r, "/testResult1", http.StatusTemporaryRedirect)
}
func (s *server) SubmitTest2(w http.ResponseWriter, r *http.Request) {
	has_user := false
	has_results := false
	test := s.makeTest(2)
	token, _ := s.getToken(r)
	results := s.getResultsPattern(test.Id)
	var user_result Result
	user_result.User_id = 0
	var new_user User
	var user User
	user.Token = ""
	rows, err := s.db.Query("SELECT * FROM users WHERE token = ($1)",
		token,
	)
	if err != nil {
		log.Fatal(err)
	}
	if rows != nil {
		for rows.Next() {
			rows.Scan(&user.Id, &user.Token)
		}
	}
	if user.Token != "" {
		has_user = true
	}
	if !has_user {
		_, err = s.db.Exec("INSERT INTO users(token) VALUES ($1)", token)
		if err != nil {
			log.Fatal(err)
		}
		rows1, err1 := s.db.Query("SELECT * FROM users WHERE token = ($1)", token)
		if err1 != nil {
			log.Fatal(err1)
		}
		if rows1 != nil {
			for rows1.Next() {
				rows1.Scan(&new_user.Id, &new_user.Token)
			}
		}
		for _, result := range results {
			sum_value := 0
			for i, question := range test.Questions {
				question.Chosen_answer.Id, _ = strconv.Atoi(r.PostFormValue(strconv.Itoa(question.Id)))
				for _, answer := range question.Answers {
					if answer.Id == question.Chosen_answer.Id {
						question.Chosen_answer = answer
					}
				}
				if result.Id == question.Chosen_answer.Result_id {
					sum_value += question.Chosen_answer.Answer_value
				}
				test.Questions[i] = question
			}
			_, err = s.db.Exec("INSERT INTO results(test_id, result_type, result_value, user_id) VALUES ($1, $2, $3, $4)", test.Id, result.Result_type, sum_value, new_user.Id)
		}
	}
	if has_user {
		rows, err = s.db.Query("SELECT * FROM results WHERE user_id = ($1) AND test_id = ($2)", user.Id, test.Id)
		if err != nil {
			log.Fatal(err)
		}
		if rows != nil {
			for rows.Next() {
				rows.Scan(&user_result.Id, &user_result.Test_id, &user_result.Result_type, &user_result.Result_value, &user_result.User_id)
			}
		}
		if user_result.Test_id != 0 {
			has_results = true
		}
		if !has_results {
			for _, result := range results {
				sum_value := 0
				for i, question := range test.Questions {
					question.Chosen_answer.Id, _ = strconv.Atoi(r.PostFormValue(strconv.Itoa(question.Id)))
					for _, answer := range question.Answers {
						if answer.Id == question.Chosen_answer.Id {
							question.Chosen_answer = answer
						}
					}
					if result.Id == question.Chosen_answer.Result_id {
						sum_value += question.Chosen_answer.Answer_value
					}
					test.Questions[i] = question
				}
				_, err = s.db.Exec("INSERT INTO results(test_id, result_type, result_value, user_id) VALUES ($1, $2, $3, $4)", test.Id, result.Result_type, sum_value, user.Id)
				if err != nil {
					log.Fatal(err)
				}
			}
		}
	}
	if has_user || has_results {
		for _, result := range results {
			sum_value := 0
			for i, question := range test.Questions {
				question.Chosen_answer.Id, _ = strconv.Atoi(r.PostFormValue(strconv.Itoa(question.Id)))
				for _, answer := range question.Answers {
					if answer.Id == question.Chosen_answer.Id {
						question.Chosen_answer = answer
					}
				}
				if result.Id == question.Chosen_answer.Result_id {
					sum_value += question.Chosen_answer.Answer_value
				}
				test.Questions[i] = question
			}
			_, err = s.db.Exec("UPDATE results SET result_value = ($1) WHERE test_id = ($2) AND result_type = ($3) AND user_id = ($4)", sum_value, test.Id, result.Result_type, user.Id)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
	results = s.getResults(test.Id, token)
	http.Redirect(w, r, "/testResult2", http.StatusTemporaryRedirect)
}
func (s *server) SubmitTest3(w http.ResponseWriter, r *http.Request) {
	has_user := false
	has_results := false
	test := s.makeTest(3)
	token, _ := s.getToken(r)
	results := s.getResultsPattern(test.Id)
	var user_result Result
	user_result.User_id = 0
	var new_user User
	var user User
	user.Token = ""
	rows, err := s.db.Query("SELECT * FROM users WHERE token = ($1)",
		token,
	)
	if err != nil {
		log.Fatal(err)
	}
	if rows != nil {
		for rows.Next() {
			rows.Scan(&user.Id, &user.Token)
		}
	}
	if user.Token != "" {
		has_user = true
	}
	if !has_user {
		_, err = s.db.Exec("INSERT INTO users(token) VALUES ($1)", token)
		if err != nil {
			log.Fatal(err)
		}
		rows1, err1 := s.db.Query("SELECT * FROM users WHERE token = ($1)", token)
		if err1 != nil {
			log.Fatal(err1)
		}
		if rows1 != nil {
			for rows1.Next() {
				rows1.Scan(&new_user.Id, &new_user.Token)
			}
		}
		for _, result := range results {
			sum_value := 0
			for i, question := range test.Questions {
				question.Chosen_answer.Id, _ = strconv.Atoi(r.PostFormValue(strconv.Itoa(question.Id)))
				for _, answer := range question.Answers {
					if answer.Id == question.Chosen_answer.Id {
						question.Chosen_answer = answer
					}
				}
				if result.Id == question.Chosen_answer.Result_id {
					sum_value += question.Chosen_answer.Answer_value
				}
				test.Questions[i] = question
			}
			_, err = s.db.Exec("INSERT INTO results(test_id, result_type, result_value, user_id) VALUES ($1, $2, $3, $4)", test.Id, result.Result_type, sum_value, new_user.Id)
		}
	}
	if has_user {
		rows, err = s.db.Query("SELECT * FROM results WHERE user_id = ($1) AND test_id = ($2)", user.Id, test.Id)
		if err != nil {
			log.Fatal(err)
		}
		if rows != nil {
			for rows.Next() {
				rows.Scan(&user_result.Id, &user_result.Test_id, &user_result.Result_type, &user_result.Result_value, &user_result.User_id)
			}
		}
		if user_result.Test_id != 0 {
			has_results = true
		}
		if !has_results {
			for _, result := range results {
				sum_value := 0
				for i, question := range test.Questions {
					question.Chosen_answer.Id, _ = strconv.Atoi(r.PostFormValue(strconv.Itoa(question.Id)))
					for _, answer := range question.Answers {
						if answer.Id == question.Chosen_answer.Id {
							question.Chosen_answer = answer
						}
					}
					if result.Id == question.Chosen_answer.Result_id {
						sum_value += question.Chosen_answer.Answer_value
					}
					test.Questions[i] = question
				}
				_, err = s.db.Exec("INSERT INTO results(test_id, result_type, result_value, user_id) VALUES ($1, $2, $3, $4)", test.Id, result.Result_type, sum_value, user.Id)
				if err != nil {
					log.Fatal(err)
				}
			}
		}
	}
	if has_user || has_results {
		for _, result := range results {
			sum_value := 0
			for i, question := range test.Questions {
				question.Chosen_answer.Id, _ = strconv.Atoi(r.PostFormValue(strconv.Itoa(question.Id)))
				for _, answer := range question.Answers {
					if answer.Id == question.Chosen_answer.Id {
						question.Chosen_answer = answer
					}
				}
				if result.Id == question.Chosen_answer.Result_id {
					sum_value += question.Chosen_answer.Answer_value
				}
				test.Questions[i] = question
			}
			_, err = s.db.Exec("UPDATE results SET result_value = ($1) WHERE test_id = ($2) AND result_type = ($3) AND user_id = ($4)", sum_value, test.Id, result.Result_type, user.Id)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
	results = s.getResults(test.Id, token)
	http.Redirect(w, r, "/testResult3", http.StatusTemporaryRedirect)
}
func (s *server) TestResult1(w http.ResponseWriter, r *http.Request) {
	token, _ := s.getToken(r)
	results := s.getResults(1, token)
	t, err := template.ParseFiles("templates/testresult1.html")
	if err != nil {
		log.Fatal(err)
	}
	if err := t.Execute(w, results); err != nil {
		log.Fatal(err)
	}
}
func (s *server) TestResult2(w http.ResponseWriter, r *http.Request) {
	token, _ := s.getToken(r)
	results := s.getResults(2, token)
	t, err := template.ParseFiles("templates/testresult.html")
	if err != nil {
		log.Fatal(err)
	}
	if err := t.Execute(w, results); err != nil {
		log.Fatal(err)
	}
}
func (s *server) TestResult3(w http.ResponseWriter, r *http.Request) {
	token, _ := s.getToken(r)
	results := s.getResults(3, token)
	t, err := template.ParseFiles("templates/testresult.html")
	if err != nil {
		log.Fatal(err)
	}
	if err := t.Execute(w, results); err != nil {
		log.Fatal(err)
	}
}

func (s *server) indexPage(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("templates/index.html")
	if err != nil {
		log.Fatal(err)
	}
	if err := t.Execute(w, nil); err != nil {
		log.Fatal(err)
	}
}

func (s *server) authMiddleware(handler func(w http.ResponseWriter, r *http.Request)) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := s.getToken(r); !ok {
			s.generateToken(w, r)
		}
		handler(w, r)
	}
}

func (s *server) Tests(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("templates/tests.html")
	if err != nil {
		log.Fatal(err)
	}
	token, _ := s.getToken(r)
	tests := [3]Test{}
	for i := 0; i < 3; i++ {
		tests[i].IsCompleted = false
		if s.getResults(i+1, token) != nil {
			tests[i].IsCompleted = true
		}
	}
	if err := t.Execute(w, tests); err != nil {
		log.Fatal(err)
	}
}

func (s *server) Contacts(w http.ResponseWriter, r *http.Request) {

	t, err := template.ParseFiles("templates/contacts.html")
	if err != nil {
		log.Fatal(err)
	}
	if err := t.Execute(w, nil); err != nil {
		log.Fatal(err)
	}
}

type User struct {
	Id    int
	Token string
}

type Test struct {
	Id          int
	Label       string
	Questions   []Question
	IsCompleted bool
}

type Question struct {
	Id            int
	Test_id       int
	Question      string
	Answers       []Answer
	Chosen_answer Answer
}

type Result struct {
	Id           int
	Test_id      int
	Result_type  string
	Result_value int
	User_id      int
}

type Answer struct {
	Id           int
	Question_id  int
	Answer       string
	Answer_value int
	Result_id    int
}

func (s *server) deleteTables() {
	_, err := s.db.Exec("DROP TABLE IF EXISTS users")
	if err != nil {
		log.Fatal(err)
	}
	_, err = s.db.Exec("DROP TABLE IF EXISTS tests")
	if err != nil {
		log.Fatal(err)
	}
	_, err = s.db.Exec("DROP TABLE IF EXISTS questions")
	if err != nil {
		log.Fatal(err)
	}
	_, err = s.db.Exec("DROP TABLE IF EXISTS answers")
	if err != nil {
		log.Fatal(err)
	}
	_, err = s.db.Exec("DROP TABLE IF EXISTS results")
	if err != nil {
		log.Fatal(err)
	}
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func (s *server) getToken(r *http.Request) (string, bool) {
	cookie, err := r.Cookie("token")
	if err != nil {
		switch {
		case errors.Is(err, http.ErrNoCookie):
			return "", false
		default:
			log.Fatal(err)
		}
	}
	return cookie.Value, true
}

func (s *server) generateToken(w http.ResponseWriter, r *http.Request) {
	token := randStringRunes(16)
	cookie := http.Cookie{
		Name:     "token",
		Value:    token,
		Path:     "/",
		MaxAge:   3600,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, &cookie)
}
func (s *server) getResultsPattern(test_id int) (results []Result) {
	rows, err := s.db.Query("SELECT id ,test_id ,result_type ,result_value from results WHERE test_id = ($1) AND user_id = 0", test_id)
	if err != nil {
		log.Fatal(err)
	}
	for rows.Next() {
		var result Result
		rows.Scan(&result.Id, &result.Test_id, &result.Result_type, &result.Result_value)
		results = append(results, result)
	}
	return results
}
func (s *server) getResults(test_id int, token string) (results []Result) {
	rows, err := s.db.Query("SELECT id FROM users WHERE token = ($1)", token)
	if err != nil {
		log.Fatal(err)
	}
	user_id := -1
	for rows.Next() {
		rows.Scan(&user_id)
	}
	rows, err = s.db.Query("SELECT * from results WHERE test_id = ($1) AND user_id = ($2)", test_id, user_id)
	if err != nil {
		log.Fatal(err)
	}
	for rows.Next() {
		var result Result
		rows.Scan(&result.Id, &result.Test_id, &result.Result_type, &result.Result_value, &result.User_id)
		results = append(results, result)
	}
	return results
}

//	func processPostMiddleware(db *sql.DB) func(http.ResponseWriter, *http.Request) {
//		return func(w http.ResponseWriter, r *http.Request) {
//			processPost(db, w, r)
//		}
//	}
//
//	func (s *server.go) getAnswersMiddleware(test Test) func(http.ResponseWriter, *http.Request) {
//		return func(w http.ResponseWriter, r *http.Request) {
//			s.getTestAnswers(test, r)
//		}
//	}
func (s *server) About(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("templates/about.html")
	if err != nil {
		log.Fatal(err)
	}
	if err := t.Execute(w, nil); err != nil {
		log.Fatal(err)
	}
}

func (s *server) Recomendations(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("templates/recomendations.html")
	if err != nil {
		log.Fatal(err)
	}
	token, _ := s.getToken(r)
	profile := []int32{}
	for i := range 3 {
		results := s.getResults(i+1, token)
		if results == nil {
			http.Redirect(w, r, "/completeTests", http.StatusTemporaryRedirect)
			return
		}
		for _, result := range results {
			profile = append(profile, int32(result.Result_value))
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	rec, err := s.c.GetRecommendations(ctx, &pb.ProfileRequest{Profile: profile})
	if err != nil {
		log.Fatal("could not get recommendations: %v", err)
	}
	recommendations := rec.GetRecommendations()
	if err := t.Execute(w, recommendations); err != nil {
		log.Fatal(err)
	}
}

func (s *server) CompleteTests(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("templates/complete.html")
	if err != nil {
		log.Fatal(err)
	}
	if err := t.Execute(w, nil); err != nil {
		log.Fatal(err)
	}
}

func createTables(db *sql.DB) {
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            token TEXT
        );
		CREATE TABLE IF NOT EXISTS tests (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			label TEXT
		);
		CREATE TABLE IF NOT EXISTS questions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			test_id INTEGER,
			question TEXT,
			FOREIGN KEY (test_id) REFERENCES tests(id) ON DELETE CASCADE
		);
		CREATE TABLE IF NOT EXISTS results (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			test_id INTEGER,
			result_type TEXT,
			result_value INTEGER,
			user_id INTEGER,
			FOREIGN KEY (user_id) REFERENCES users(id),
			FOREIGN KEY (test_id) REFERENCES tests(id) ON DELETE CASCADE
		);
		CREATE TABLE IF NOT EXISTS answers (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			question_id INTEGER,
			answer TEXT,
			answer_value INTEGER,
			result_id INTEGER,
			FOREIGN KEY (question_id) REFERENCES questions(id),
			FOREIGN KEY (result_id) REFERENCES results(id)
		)
    `); err != nil {
		log.Fatal("Error creating table:", err)
	}
}

func initTables(db *sql.DB) {
	if _, err := db.Exec(`
        INSERT INTO tests(label) VALUES ('Тест на особенности характера и ценностей'), ('Тест на склонности и предпочтения'),('Тест способностей');
		INSERT INTO questions(test_id,question) VALUES 
		(1, 'Делаете ли вы необдуманные замечания  или обвинения, о которых вы после жалеете?'),
		(1, 'Считают ли другие люди ваши поступки непредсказуемыми?'),
		(1, 'Оплачиваете ли вы свои долги и сдерживаете ли вы свои обещания, если это возможно?'),
		(1, 'Достаточно  ли  хорошо  вы  справляетесь  с  проблемами повседневной жизни?'),
		(1, 'Беспокоят ли вас до сих пор прошлые неудачи?'),
		(1, 'Часто ли вы расстраиваетесь  из-за судьбы жертв войны и политических беженцев?'),
		(1, 'Доставляет ли вам  удовольствие деятельность, выбранная самостоятельно?'),
		(1, 'Трудно ли вам рассматривать предмет самоубийства?'),
		(1, 'Есть  ли у  вас привычка  грызть свои  ногти или кончик
     карандаша?'),
		(1, 'Могут  ли   некоторые  звуки  вызывать   у  вас  крайне
     неприятные ощущения, такие, будто "зубы сводит"?'),
		(1, 'В то  время,  когда  другие  начинают  терять терпение, остаетесь ли вы достаточно спокойным?'),
		(1, 'Обычно  вас  не  беспокоит  "полная  тишина",  когда вы
     хотите отдохнуть?'),
		(1, 'Сильно ли  вас беспокоит мысль  о том, что  надо начать
     какое-то новое дело?'),
		(1, 'Бывает ли у вас ощущение, что вам все снится, когда все
     в жизни кажется нереальным?'),
		(1, 'Можете  ли  вы  не  вмешиваться  при  окончании решения
     кроссворда другим человеком?'),
		(1, 'Работаете   ли   вы   "рывками",   будучи  относительно
     пассивным,  а   затем  чрезмерно  активным   в  течение
     одного-двух дней?'),
		(1, 'Ждете  ли   вы  обычно,  чтобы   другой  человек  начал
     разговор?'),
		(1, 'Вы  предпочитаете  находиться   в  ситуации,  когда  не
     приходится отвечать за принятие решений?'),
		(1, 'Можете ли вы быть "заводилой" на вечеринках?'),
		(1, 'Просматриваете ли вы расписания движения поездов, телефонные справочники или словари ради удовольствия?'),
		(1, 'Отказываетесь  ли  вы  от  ответственности  за что-либо
     из-за  ваших   сомнений,  что  вы  можете   с  этим  не
     справиться?'),
		(1, 'Имеете ли вы склонность прятать свои чувства?'),
		(1, 'Если  бы вы  подумали, что  кто-то относится  к вам и к
     вашим действиям  с подозрением, стали бы  вы выяснять с
     ними  этот вопрос  вместо того,  чтобы предоставить  им
     самим во всем разобраться?'),
		(1, 'Считаете  ли вы  себя  энергичным  в вашем  отношении к
     жизни?'),
		(1, 'Если вас попросят принять какое-то решение, может ли вас поколебать ваше чувство расположения или нерасположения по отношению к  человеку, о котором идет речь?'),
		(1, 'Когда  вы  слушаете  лектора,  посещает  ли  вас иногда
     мысль, что лектор обращается исключительно к вам?'),
		(1, 'Можете  ли вы  положиться на  то, что  подсказывает ваш
     здравый  смысл в  эмоциональной ситуации,  в которую вы
     попали?'),
		(1, 'Можете ли вы понять точку зрения другого человека, если
     вы этого захотите?'),
		(1, 'Считаете   ли   вы,   что   есть   люди,  которые  явно
     недружелюбны по отношению к вам и "копают" под вас?'),
		(1, 'Доставляет  ли  вам   удовольствие  рассказывать  самые
     свежие сплетни о ваших знакомых?'),
		(1, 'Правда ли, что вы  редко подвергаете сомнениям поступки
     других людей?'),
		(1, 'Считаетесь ли вы с  лучшими сторонами большинства людей
     и лишь редко отзываетесь неуважительно о них?'),
		(1, 'Если бы вы увидели в магазине товар, на котором явно по
     ошибке проставлена более низкая  цена, попытались бы вы
     приобрести его по этой цене?'),
		(1, 'Являетесь ли  вы сторонником разделения  людей по цвету
     кожи и классовой принадлежности?'),
		(1, 'Верно   ли,   что   вы   предпочитаете   действовать  в
     соответствии  с  пожеланиями   других,  чем  стремитесь
     делать все по-своему?'),
		(1, 'Бываете ли вы обычно правдивы с окружающими?'),
		(1, 'верно ли  то, что вас  сильно волнуют только  некоторые
     определенные темы?'),
		(1, ' Правда ли, что есть всего несколько человек, которых вы
     любите по-настоящему?'),
		(1, 'Обращаются  ли к  вам  за  помощью или  советом "просто
     знакомые" со своими личными проблемами?'),
		(1, 'Рассказывая какой-нибудь забавный  случай, можете ли вы
     легко  подражать  манерам  или  речи  участников  этого
     случая?'),
		(2, 'Выберите вид деятельности, который вам больше нравится, подходит или которым было бы интересно заниматься'),
		(2, 'Выберите вид деятельности, который вам больше нравится, подходит или которым было бы интересно заниматься'),
		(2, 'Выберите вид деятельности, который вам больше нравится, подходит или которым было бы интересно заниматься'),
		(2, 'Выберите вид деятельности, который вам больше нравится, подходит или которым было бы интересно заниматься'),
		(2, 'Выберите вид деятельности, который вам больше нравится, подходит или которым было бы интересно заниматься'),
		(2, 'Выберите вид деятельности, который вам больше нравится, подходит или которым было бы интересно заниматься'),
		(2, 'Выберите вид деятельности, который вам больше нравится, подходит или которым было бы интересно заниматься'),
		(2, 'Выберите вид деятельности, который вам больше нравится, подходит или которым было бы интересно заниматься'),
		(2, 'Выберите вид деятельности, который вам больше нравится, подходит или которым было бы интересно заниматься'),
		(2, 'Выберите вид деятельности, который вам больше нравится, подходит или которым было бы интересно заниматься'),
		(2, 'Выберите вид деятельности, который вам больше нравится, подходит или которым было бы интересно заниматься'),
		(2, 'Выберите вид деятельности, который вам больше нравится, подходит или которым было бы интересно заниматься'),
		(2, 'Выберите вид деятельности, который вам больше нравится, подходит или которым было бы интересно заниматься'),
		(2, 'Выберите вид деятельности, который вам больше нравится, подходит или которым было бы интересно заниматься'),
		(2, 'Выберите вид деятельности, который вам больше нравится, подходит или которым было бы интересно заниматься'),
		(2, 'Из представленных умений, способностей, навыков выберите то, чем обладаете или что способны осуществить'),
		(2, 'Из представленных умений, способностей, навыков выберите то, чем обладаете или что способны осуществить'),
		(2, 'Из представленных умений, способностей, навыков выберите то, чем обладаете или что способны осуществить'),
		(2, 'Из представленных умений, способностей, навыков выберите то, чем обладаете или что способны осуществить'),
		(2, 'Из представленных умений, способностей, навыков выберите то, чем обладаете или что способны осуществить'),
		(2, 'Из представленных умений, способностей, навыков выберите то, чем обладаете или что способны осуществить'),
		(2, 'Из представленных умений, способностей, навыков выберите то, чем обладаете или что способны осуществить'),
		(2, 'Из представленных умений, способностей, навыков выберите то, чем обладаете или что способны осуществить'),
		(2, 'Из представленных умений, способностей, навыков выберите то, чем обладаете или что способны осуществить'),
		(2, 'Из представленных умений, способностей, навыков выберите то, чем обладаете или что способны осуществить'),
		(2, 'Из представленных умений, способностей, навыков выберите то, чем обладаете или что способны осуществить'),
		(2, 'Из представленных умений, способностей, навыков выберите то, чем обладаете или что способны осуществить'),
		(2, 'Из представленных умений, способностей, навыков выберите то, чем обладаете или что способны осуществить'),
		(2, 'Из представленных умений, способностей, навыков выберите то, чем обладаете или что способны осуществить'),
		(2, 'Из представленных умений, способностей, навыков выберите то, чем обладаете или что способны осуществить'),
		(2, 'Выберите профессию которая вам нравится, интересна'),
		(2, 'Выберите профессию которая вам нравится, интересна'),
		(2, 'Выберите профессию которая вам нравится, интересна'),
		(2, 'Выберите профессию которая вам нравится, интересна'),
		(2, 'Выберите профессию которая вам нравится, интересна'),
		(2, 'Выберите профессию которая вам нравится, интересна'),
		(2, 'Выберите профессию которая вам нравится, интересна'),
		(2, 'Выберите профессию которая вам нравится, интересна'),
		(2, 'Выберите профессию которая вам нравится, интересна'),
		(2, 'Выберите профессию которая вам нравится, интересна'),
		(2, 'Выберите профессию которая вам нравится, интересна'),
		(2, 'Выберите профессию которая вам нравится, интересна'),
		(2, 'Выберите профессию которая вам нравится, интересна'),
		(2, 'Выберите профессию которая вам нравится, интересна'),
		(2, 'Выберите профессию которая вам нравится, интересна'),
		(2, 'Выберите профессию которая вам нравится, интересна'),
		(2, 'Выберите профессию которая вам нравится, интересна'),
		(2, 'Выберите профессию которая вам нравится, интересна'),
		(2, 'Выберите профессию которая вам нравится, интересна'),
		(2, 'Выберите профессию которая вам нравится, интересна'),
		(2, 'Выберите профессию которая вам нравится, интересна'),
		(2, 'Выберите профессию которая вам нравится, интересна'),
		(2, 'Выберите профессию которая вам нравится, интересна'),
		(2, 'Выберите профессию которая вам нравится, интересна'),
		(2, 'Выберите профессию которая вам нравится, интересна'),
		(2, 'Выберите профессию которая вам нравится, интересна'),
		(2, 'Выберите профессию которая вам нравится, интересна'),
		(2, 'Выберите профессию которая вам нравится, интересна'),
		(2, 'Выберите профессию которая вам нравится, интересна'),
		(2, 'Выберите профессию которая вам нравится, интересна'),
		(3, 'Верно ли, что в детстве ты очень любил подолгу играть в подвижные игры? '),
		(3, 'Верно ли, что в детстве ты очень любил придумывать игры и верховодить в них?'),
		(3, 'Верно ли, что в детстве ты очень любил играть в шашки, шахматы?'),
		(3, 'Верно ли, что в детстве ты очень любил ломать игрушки, чтобы посмотреть, что внутри?'),
		(3, 'Верно ли, что в детстве ты очень любил читать стихи или петь песни?'),
		(3, 'Верно ли, что в детстве ты очень любил разговаривать с незнакомыми или задавать вопросы?'),
		(3, 'Верно ли, что в детстве ты очень любил слушать музыку и ритмично танцевать под нее?'),
		(3, 'Верно ли, что в детстве ты очень любил рисовать сам или наблюдать, как рисуют другие?'),
		(3, 'Верно ли, что в детстве ты очень любил слушать или сочинять сказки или истории?'),
		(3, 'Нравится ли тебе сейчас заниматься на уроках физкультуры или в спортшколе, секции?'),
		(3, 'Нравится ли тебе сейчас добровольно брать на себя обязанности организатора дела?'),
		(3, 'Нравится ли тебе сейчас помогать ребятам решать математические задачи?'),
		(3, 'Нравится ли тебе сейчас читать об известных открытиях и изобретениях?'),
		(3, 'Нравится ли тебе сейчас участвовать в художественной самодеятельности?'),
		(3, 'Нравится ли тебе сейчас помогать другим людям разбираться в их проблемах?'),
		(3, 'Нравится ли тебе сейчас читать или узнавать что-то новое об искусстве?'),
		(3, 'Нравится ли тебе сейчас заниматься в изостудии, изокружке?'),
		(3, 'Нравится ли тебе сейчас писать сочинения на свободную тему?'),
		(3, 'Часто ли тебя тянет к длительным физическим упражнениям?'),
		(3, 'Часто ли тебя тянет к делам в группе, требующим твоей инициативы или настойчивости?'),
		(3, 'Часто ли тебя тянет к разгадыванию математических шарад?'),
		(3, 'Часто ли тебя тянет к изготовлению каких-нибудь изделий (моделей)?'),
		(3, 'Часто ли тебя тянет участвовать в постановке спектакля?'),
		(3, 'Часто ли тебя тянет помочь людям, посочувствовать им?'),
		(3, 'Часто ли тебя тянет поиграть на музыкальном инструменте?'),
		(3, 'Часто ли тебя тянет порисовать красками или карандашами?'),
		(3, 'Часто ли тебя тянет писать стихи, прозу или просто вести дневник?'),
		(3, 'Любишь ли ты долгое время заниматься спортом или физическим трудом?'),
		(3, 'Любишь ли ты долгое время энергично работать вместе с другими?'),
		(3, 'Любишь ли ты долгое время заниматься черчением или шахматной комбинацией?'),
		(3, 'Любишь ли ты долгое время копаться в механизмах, приборах?'),
		(3, 'Любишь ли ты долгое время заботиться о младших, слабых или больных людях?'),
		(3, 'Любишь ли ты долгое время думать над судьбами людей, героев понравившихся книг?'),
		(3, 'Любишь ли ты долгое время исполнять музыкальные пьесы?'),
		(3, 'Любишь ли ты долгое время рисовать, лепить, фантазируя при этом?'),
		(3, 'Любишь ли ты долгое время готовиться к докладу, сообщению, сочинению?');
		INSERT INTO answers(question_id, answer, answer_value, result_id) VALUES
		(1, 'да', 6, '1'), (1, 'скорее да', 5, '1'), (1, 'сомневаюсь', 4, '1'), (1, 'скорее нет', 3, '1'), (1, 'нет', 2, '1'),
		(2, 'да', 6, '1'), (2, 'скорее да', 5, '1'), (2, 'сомневаюсь', 4, '1'), (2, 'скорее нет', 3, '1'), (2, 'нет', 2, '1'),
		(3, 'да', 2, '1'), (3, 'скорее да', 3, '1'), (3, 'сомневаюсь', 4, '1'), (3, 'скорее нет', 5, '1'), (3, 'нет', 6, '1'),
		(4, 'да', 2, '1'), (4, 'скорее да', 3, '1'), (4, 'сомневаюсь', 4, '1'), (4, 'скорее нет', 5, '1'), (4, 'нет', 6, '1'),
		(5, 'да', 6, '2'), (5, 'скорее да', 5, '2'), (5, 'сомневаюсь', 4, '2'), (5, 'скорее нет', 3, '2'), (5, 'нет', 2, '2'),
		(6, 'да', 6, '2'), (6, 'скорее да', 5, '2'), (6, 'сомневаюсь', 4, '2'), (6, 'скорее нет', 3, '2'), (6, 'нет', 2, '2'),
		(7, 'да', 2, '2'), (7, 'скорее да', 3, '2'), (7, 'сомневаюсь', 4, '2'), (7, 'скорее нет', 5, '2'), (7, 'нет', 6, '2'),
		(8, 'да', 2, '2'), (8, 'скорее да', 3, '2'), (8, 'сомневаюсь', 4, '2'), (8, 'скорее нет', 5, '2'), (8, 'нет', 6, '2'),
		(9, 'да', 6, '3'), (9, 'скорее да', 5, '3'), (9, 'сомневаюсь', 4, '3'), (9, 'скорее нет', 3, '3'), (9, 'нет', 2, '3'),
		(10, 'да', 6, '3'), (10, 'скорее да', 5, '3'), (10, 'сомневаюсь', 4, '3'), (10, 'скорее нет', 3, '3'), (10, 'нет', 2, '3'),
		(11, 'да', 2, '3'), (11, 'скорее да', 3, '3'), (11, 'сомневаюсь', 4, '3'), (11, 'скорее нет', 5, '3'), (11, 'нет', 6, '3'),
		(12, 'да', 2, '3'), (12, 'скорее да', 3, '3'), (12, 'сомневаюсь', 4, '3'), (12, 'скорее нет', 5, '3'), (12, 'нет', 6, '3'),
		(13, 'да', 6, '4'), (13, 'скорее да', 5, '4'), (13, 'сомневаюсь', 4, '4'), (13, 'скорее нет', 3, '4'), (13, 'нет', 2, '4'),
		(14, 'да', 6, '4'), (14, 'скорее да', 5, '4'), (14, 'сомневаюсь', 4, '4'), (14, 'скорее нет', 3, '4'), (14, 'нет', 2, '4'),
		(15, 'да', 2, '4'), (15, 'скорее да', 3, '4'), (15, 'сомневаюсь', 4, '4'), (15, 'скорее нет', 5, '4'), (15, 'нет', 6, '4'),
		(16, 'да', 6, '4'), (16, 'скорее да', 5, '4'), (16, 'сомневаюсь', 4, '4'), (16, 'скорее нет', 3, '4'), (16, 'нет', 2, '4'),
		(17, 'да', 6, '5'), (17, 'скорее да', 5, '5'), (17, 'сомневаюсь', 4, '5'), (17, 'скорее нет', 3, '5'), (17, 'нет', 2, '5'),
		(18, 'да', 6, '5'), (18, 'скорее да', 5, '5'), (18, 'сомневаюсь', 4, '5'), (18, 'скорее нет', 3, '5'), (18, 'нет', 2, '5'),
		(19, 'да', 2, '5'), (19, 'скорее да', 3, '5'), (19, 'сомневаюсь', 4, '5'), (19, 'скорее нет', 5, '5'), (19, 'нет', 6, '5'),
		(20, 'да', 2, '5'), (20, 'скорее да', 3, '5'), (20, 'сомневаюсь', 4, '5'), (20, 'скорее нет', 5, '5'), (20, 'нет', 6, '5'),
		(21, 'да', 6, '6'), (21, 'скорее да', 5, '6'), (21, 'сомневаюсь', 4, '6'), (21, 'скорее нет', 3, '6'), (21, 'нет', 2, '6'),
		(22, 'да', 6, '6'), (22, 'скорее да', 5, '6'), (22, 'сомневаюсь', 4, '6'), (22, 'скорее нет', 3, '6'), (22, 'нет', 2, '6'),
		(23, 'да', 2, '6'), (23, 'скорее да', 3, '6'), (23, 'сомневаюсь', 4, '6'), (23, 'скорее нет', 5, '6'), (23, 'нет', 6, '6'),
		(24, 'да', 2, '6'), (24, 'скорее да', 3, '6'), (24, 'сомневаюсь', 4, '6'), (24, 'скорее нет', 5, '6'), (24, 'нет', 6, '6'),
		(25, 'да', 6, '7'), (25, 'скорее да', 5, '7'), (25, 'сомневаюсь', 4, '7'), (25, 'скорее нет', 3, '7'), (25, 'нет', 2, '7'),
		(26, 'да', 6, '7'), (26, 'скорее да', 5, '7'), (26, 'сомневаюсь', 4, '7'), (26, 'скорее нет', 3, '7'), (26, 'нет', 2, '7'),
		(27, 'да', 2, '7'), (27, 'скорее да', 3, '7'), (27, 'сомневаюсь', 4, '7'), (27, 'скорее нет', 5, '7'), (27, 'нет', 6, '7'),
		(28, 'да', 2, '7'), (28, 'скорее да', 3, '7'), (28, 'сомневаюсь', 4, '7'), (28, 'скорее нет', 5, '7'), (28, 'нет', 6, '7'),
		(29, 'да', 6, '8'), (29, 'скорее да', 5, '8'), (29, 'сомневаюсь', 4, '8'), (29, 'скорее нет', 3, '8'), (29, 'нет', 2, '8'),
		(30, 'да', 6, '8'), (30, 'скорее да', 5, '8'), (30, 'сомневаюсь', 4, '8'), (30, 'скорее нет', 3, '8'), (30, 'нет', 2, '8'),
		(31, 'да', 2, '8'), (31, 'скорее да', 3, '8'), (31, 'сомневаюсь', 4, '8'), (31, 'скорее нет', 5, '8'), (31, 'нет', 6, '8'),
		(32, 'да', 2, '8'), (32, 'скорее да', 3, '8'), (32, 'сомневаюсь', 4, '8'), (32, 'скорее нет', 5, '8'), (32, 'нет', 6, '8'),
		(33, 'да', 6, '9'), (33, 'скорее да', 5, '9'), (33, 'сомневаюсь', 4, '9'), (33, 'скорее нет', 3, '9'), (33, 'нет', 2, '9'),
		(34, 'да', 6, '9'), (34, 'скорее да', 5, '9'), (34, 'сомневаюсь', 4, '9'), (34, 'скорее нет', 3, '9'), (34, 'нет', 2, '9'),
		(35, 'да', 2, '9'), (35, 'скорее да', 3, '9'), (35, 'сомневаюсь', 4, '9'), (35, 'скорее нет', 5, '9'), (35, 'нет', 6, '9'),
		(36, 'да', 2, '9'), (36, 'скорее да', 3, '9'), (36, 'сомневаюсь', 4, '9'), (36, 'скорее нет', 5, '9'), (36, 'нет', 6, '9'),
		(37, 'да', 6, '10'), (37, 'скорее да', 5, '10'), (37, 'сомневаюсь', 4, '10'), (37, 'скорее нет', 3, '10'), (37, 'нет', 2, '10'),
		(38, 'да', 6, '10'), (38, 'скорее да', 5, '10'), (38, 'сомневаюсь', 4, '10'), (38, 'скорее нет', 3, '10'), (38, 'нет', 2, '10'),
		(39, 'да', 2, '10'), (39, 'скорее да', 3, '10'), (39, 'сомневаюсь', 4, '10'), (39, 'скорее нет', 5, '10'), (39, 'нет', 6, '10'),
		(40, 'да', 2, '10'), (40, 'скорее да', 3, '10'), (40, 'сомневаюсь', 4, '10'), (40, 'скорее нет', 5, '10'), (40, 'нет', 6, '10'),
		(41, 'Пройти курс обучения работам по дереву', 1, '11'), (41, 'Работать в научно-исследовательской лаборатории', 1, '12'),
		(42, 'Работать на легковом автомобиле', 1, '11'), (42, 'Играть на музыкальном инструменте', 1, '13'),
		(43, 'Ремонтировать хозяйственные постройки', 1, '11'), (43, 'Работать в сфере социальной поддержки и защиты', 1, '14'),
		(44, 'Ремонтировать электроприборы', 1, '11'), (44, 'Быть руководителем какого-либо проекта или мероприятия', 1, '15'),
		(45, 'Настраивать музыкальную стереосистему', 1, '11'), (45, 'Содержать свой рабочий стол и служебное помещение в порядке', 1, '16'),
		(46, 'Применять матетатику для решения практических задач', 1, '12'), (46, 'Писать статьи для журнала или газеты', 1, '13'),
		(47, 'Изучать научные теории', 1, '12'), (47, 'Обучаться на курсах психологии', 1, '14'),
		(48, 'Анализировать информацию для разработки новых предложений и рекомендаций', 1, '12'), (48, 'Пройти курсы или семинар для руководителей, менеджеров', 1, '15'),
		(49, 'Читать научные книги и журналы', 1, '12'), (49, 'Проводить инвентаризацию материальных ресурсов', 1, '16'),
		(50, 'Воплощать в драматическое произведение рассказ или художественный замысел', 1, '13'), (50, 'Изучать факты нарушения закона несовершеннолетними', 1, '14'),
		(51, 'Играть в ансамбле, группе или оркестре', 1, '13'), (51, 'Читать о руководителях в бизнесе или правительстве', 1, '15'),
		(52, 'Конструировать мебель или одежду', 1, '13'), (52, 'Работать с компьютером', 1, '16'),
		(53, 'Дискутировать по вопросам взаимоотношений между людьми', 1, '14'), (53, 'Участвовать в политических компаниях', 1, '15'),
		(54, 'Обучать других выполнять какую-либо работу', 1, '14'), (54, 'Вести учет своих доходов и расходов', 1, '16'),
		(55, 'Организовать собственное дело и управлять им', 1, '15'), (55, 'Проводить проверку документации или продукции на предмет выявления ошибок или брака', 1, '16'),
		(56, 'Выполнять простой ремонт телевизора, радиоприемника', 1, '11'), (56, 'Использовать компьютер при изучении научной проблемы', 1, '12'),
		(57, 'Ремонтировать мебель', 1, '11'), (57, 'Написать рассказ', 1, '13'),
		(58, 'Использовать столярные инструменты для работ по дереву', 1, '11'), (58, 'Уверенно помогать другим в принятии решений', 1, '14'),
		(59, 'Читать чертежи, эскизы, схемы', 1, '11'), (59, 'Организовать работу других', 1, '15'),
		(60, 'Провести электрическую проводку в помещении', 1, '11'), (60, 'Обрабатывать корреспонденцию и другие документы', 1, '16'),
		(61, 'Разобраться в физических свойствах многих веществ', 1, '12'), (61, 'Создать рекламный плакат', 1, '13'),
		(62, 'Расшифровать простые химические формулы', 1, '12'), (62, 'Помогать людям, страдающим физическими недостатками', 1, '14'),
		(63, 'Объяснить причины болезни человека', 1, '12'), (63, 'Объективно оценить собственные достоинства, возможности', 1, '15'),
		(64, 'Использовать математическую статистику для решения научных проблем', 1, '12'), (64, 'Легко получить необходимую информацию по телефону', 1, '16'),
		(65, 'Писать красками, акварелью, лепить скульптуру', 1, '13'), (65, 'Выполнять роль хозяина, тамады на праздничных вечеринках', 1, '14'),
		(66, 'Обрисовать или описать человека так, что его можно было узнать', 1, '13'), (66, 'Легко заинтересовать других', 1, '15'),
		(67, 'Создать сценическое воплощение идеи или сюжета', 1, '13'), (67, 'Вести точный учет доходов и расходов', 1, '16'),
		(68, 'Доступно объяснять какие-либо вещи другим', 1, '14'), (68, 'Организовать м управлять компанией по продаже', 1, '15'),
		(69, 'Возглавить групповую дискуссию', 1, '14'), (69, 'Использовать компьютер для анализа данных бизнеса', 1, '16'),
		(70, 'Успешно торговать чем-либо', 1, '15'), (70, 'Быстро и без ошибок напечатать текст', 1, '16'),
		(71, 'Плотник', 1, '11'), (71, 'Инженер-конструктор', 1, '12'),
		(72, 'Фермер', 1, '11'), (72, 'Писатель', 1, '13'),
		(73, 'Автослесарь', 1, '11'), (73, 'Преподаватель высшей школы', 1, '14'),
		(74, 'Специалист по электронной аппаратуре', 1, '11'), (74, 'Управляющий фирмой', 1, '15'),
		(75, 'Лесник', 1, '11'), (75, 'Экономист', 1, '16'),
		(76, 'Шофер', 1, '11'), (76, 'Техник медицинской лаборатории', 1, '12'),
		(77, 'Сварщик', 1, '11'), (77, 'Фотограф', 1, '13'),
		(78, 'Радиоинженер', 1, '11'), (78, 'Сотрудник службы социальной поддержки', 1, '14'),
		(79, 'Гравировщик, изготовитель печатей, штампов', 1, '11'), (79, 'Управляющий гостиницей', 1, '15'),
		(80, 'Экономист-плановик производства', 1, '11'), (80, 'Счетовод', 1, '16'),
		(81, 'Физик', 1, '12'), (81, 'Музыкант-аранжировщик', 1, '13'),
		(82, 'Химик', 1, '12'), (82, 'Логопед', 1, '14'),
		(83, 'Издатель научного или научно-популярного журнала', 1, '12'), (83, 'Директор на радио или телевидении', 1, '15'),
		(84, 'Ботаник', 1, '12'), (84, 'Секретарь-референт', 1, '16'),
		(85, 'Хирург', 1, '12'), (85, 'Художник', 1, '13'),
		(86, 'Антрополог', 1, '12'), (86, 'Учитель школы', 1, '14'),
		(87, 'Терапевт', 1, '12'), (87, 'Агент по продаже недвижимости', 1, '15'),
		(88, 'Метеоролог', 1, '12'), (88, 'Кассир в банке', 1, '16'),
		(89, 'Певец', 1, '13'), (89, 'Психолог', 1, '14'),
		(90, 'Автор художественных произведений', 1, '13'), (90, 'Страховой агент', 1, '15'),
		(91, 'Музыкант-исполнитель', 1, '13'), (91, 'Налоговый инспектор', 1, '16'),
		(92, 'Эксперт по живописи', 1, '13'), (92, 'Специалист по семейному консультированию', 1, '14'),
		(93, 'Журналист', 1, '13'), (93, 'Заведующий отделом маркетинга', 1, '15'),
		(94, 'Модельер одежды', 1, '13'), (94, 'Ревизор', 1, '16'),
		(95, 'Инструктор молодежного лагеря', 1, '14'), (95, 'Управляющий магазином', 1, '15'),
		(96, 'Консультант по выбору профессии', 1, '14'), (96, 'Переводчик текстов', 1, '16'),
		(97, 'Социолог', 1, '14'), (97, 'Адвокат', 1, '15'),
		(98, 'Инспектор по делам несовершеннолетних', 1, '14'), (98, 'Оператор ПК', 1, '16'),
		(99, 'Рекламный агент', 1, '15'), (99, 'Инспектор в банке', 1, '16'),
		(100, 'Посредник в торговых операциях', 1, '15'), (100, 'Судебный исполнитель', 1, '16'),
		(101, 'Да', 1, '17'), (101, 'Нет', 0, '17'),
		(102, 'Да', 1, '18'), (102, 'Нет', 0, '18'),
		(103, 'Да', 1, '19'), (103, 'Нет', 0, '19'),
		(104, 'Да', 1, '20'), (104, 'Нет', 0, '20'),
		(105, 'Да', 1, '21'), (105, 'Нет', 0, '21'),
		(106, 'Да', 1, '22'), (106, 'Нет', 0, '22'),
		(107, 'Да', 1, '23'), (107, 'Нет', 0, '23'),
		(108, 'Да', 1, '24'), (108, 'Нет', 0, '24'),
		(109, 'Да', 1, '25'), (109, 'Нет', 0, '25'),
		(110, 'Да', 1, '17'), (110, 'Нет', 0, '17'),
		(111, 'Да', 1, '18'), (111, 'Нет', 0, '18'),
		(112, 'Да', 1, '19'), (112, 'Нет', 0, '19'),
		(113, 'Да', 1, '20'), (113, 'Нет', 0, '20'),
		(114, 'Да', 1, '21'), (114, 'Нет', 0, '21'),
		(115, 'Да', 1, '22'), (115, 'Нет', 0, '22'),
		(116, 'Да', 1, '23'), (116, 'Нет', 0, '23'),
		(117, 'Да', 1, '24'), (117, 'Нет', 0, '24'),
		(118, 'Да', 1, '25'), (118, 'Нет', 0, '25'),
		(119, 'Да', 1, '17'), (119, 'Нет', 0, '17'),
		(120, 'Да', 1, '18'), (120, 'Нет', 0, '18'),
		(121, 'Да', 1, '19'), (121, 'Нет', 0, '19'),
		(122, 'Да', 1, '20'), (122, 'Нет', 0, '20'),
		(123, 'Да', 1, '21'), (123, 'Нет', 0, '21'),
		(124, 'Да', 1, '22'), (124, 'Нет', 0, '22'),
		(125, 'Да', 1, '23'), (125, 'Нет', 0, '23'),
		(126, 'Да', 1, '24'), (126, 'Нет', 0, '24'),
		(127, 'Да', 1, '25'), (127, 'Нет', 0, '25'),
		(128, 'Да', 1, '17'), (128, 'Нет', 0, '17'),
		(129, 'Да', 1, '18'), (129, 'Нет', 0, '18'),
		(130, 'Да', 1, '19'), (130, 'Нет', 0, '19'),
		(131, 'Да', 1, '20'), (131, 'Нет', 0, '20'),
		(132, 'Да', 1, '21'), (132, 'Нет', 0, '21'),
		(133, 'Да', 1, '22'), (133, 'Нет', 0, '22'),
		(134, 'Да', 1, '23'), (134, 'Нет', 0, '23'),
		(135, 'Да', 1, '24'), (135, 'Нет', 0, '24'),
		(136, 'Да', 1, '25'), (136, 'Нет', 0, '25'),
		(137, 'Да', 1, '17'), (137, 'Нет', 0, '17'),
		(138, 'Да', 1, '18'), (138, 'Нет', 0, '18'),
		(139, 'Да', 1, '19'), (139, 'Нет', 0, '19'),
		(140, 'Да', 1, '20'), (140, 'Нет', 0, '20'),
		(141, 'Да', 1, '21'), (141, 'Нет', 0, '21'),
		(142, 'Да', 1, '22'), (142, 'Нет', 0, '22'),
		(143, 'Да', 1, '23'), (143, 'Нет', 0, '23'),
		(144, 'Да', 1, '24'), (144, 'Нет', 0, '24'),
		(145, 'Да', 1, '25'), (145, 'Нет', 0, '25');
		INSERT INTO results(test_id, result_type, result_value, user_id) VALUES 
		(1,'A',0, 0), (1,'B',0, 0), (1,'C',0, 0), (1,'D',0, 0), (1,'E',0, 0), (1,'F',0, 0), (1,'G',0, 0), (1,'H',0, 0), (1,'I',0, 0), (1,'J',0, 0),
		(2, 'R', 0, 0), (2, 'I', 0, 0), (2, 'A', 0, 0), (2, 'S', 0, 0), (2, 'E', 0, 0), (2, 'C', 0, 0),
		(3,'Физические', 0,0), (3,'Организаторские', 0,0), (3,'Математические', 0,0), (3,'Конструкторско-технические', 0,0), (3,'Артистические', 0,0), (3,'Коммуникативные', 0,0), (3,'Музыкальные', 0,0), (3,'Художественно-изобразительные', 0,0), (3,'Филологические', 0,0);
    `); err != nil {
		log.Fatal("Error creating table:", err)
	}
}

//func (s *server.go) register(w http.ResponseWriter, r *http.Request){
//
//}

func main() {
	db, err := sql.Open("sqlite3", "test.db")
	if err != nil {
		log.Fatal("Error opening database:", err)
	}
	defer db.Close()
	createTables(db)
	conn, err := grpc.NewClient("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewProfileServiceClient(conn)
	//initTables(db)

	s := server{db, c}
	//s.deleteTables()

	http.HandleFunc("/", s.authMiddleware(s.indexPage))
	http.HandleFunc("/tests", s.authMiddleware(s.Tests))
	http.HandleFunc("/contacts", s.authMiddleware(s.Contacts))
	http.HandleFunc("/about", s.authMiddleware(s.About))
	http.HandleFunc("/test1", s.authMiddleware(s.Test1))
	http.HandleFunc("/test2", s.authMiddleware(s.Test2))
	http.HandleFunc("/test3", s.authMiddleware(s.Test3))
	http.HandleFunc("/completeTests", s.authMiddleware(s.CompleteTests))
	http.HandleFunc("/recomendations", s.authMiddleware(s.Recomendations))
	http.HandleFunc("/SubmitTest1", s.authMiddleware(s.SubmitTest1))
	http.HandleFunc("/SubmitTest2", s.authMiddleware(s.SubmitTest2))
	http.HandleFunc("/SubmitTest3", s.authMiddleware(s.SubmitTest3))
	http.HandleFunc("/testResult1", s.authMiddleware(s.TestResult1))
	http.HandleFunc("/testResult2", s.authMiddleware(s.TestResult2))
	http.HandleFunc("/testResult3", s.authMiddleware(s.TestResult3))

	http.ListenAndServe(":80", nil)
}
