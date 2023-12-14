package main

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
	"regexp"
	"runtime"
	"siam/helper"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type JUserLoginRequest struct {
	Id              int
	UsernameMaster  string
	UsernameSession string
	Password        string
	Nama            string
	Role            string
	Status          string
	ParamKey        string
	Method          string
	Page            int
	RowPage         int
	OrderBy         string
	Order           string
}

type JUserLoginResponse struct {
	Id           int
	Username     string
	Password     string
	Nama         string
	Role         string
	Status       int
	TanggalInput string
}

func UserLogin(c *gin.Context) {
	db := helper.Connect(c)
	defer db.Close()
	StartTime := time.Now()
	StartTimeStr := StartTime.String()
	PageGo := "USER_LOGIN"

	var (
		bodyBytes    []byte
		XRealIp      string
		IP           string
		LogFile      string
		totalPage    float64
		totalRecords float64
	)

	jUserLoginRequest := JUserLoginRequest{}
	jUserLoginResponse := JUserLoginResponse{}
	jUserLoginResponses := []JUserLoginResponse{}

	AllHeader := helper.ReadAllHeader(c)
	LogFile = os.Getenv("LOGFILE")
	Method := c.Request.Method
	Path := c.Request.URL.EscapedPath()

	// ---------- start get ip ----------
	if Values, _ := c.Request.Header["X-Real-Ip"]; len(Values) > 0 {
		XRealIp = Values[0]
	}

	if XRealIp != "" {
		IP = XRealIp
	} else {
		IP = c.ClientIP()
	}
	// ---------- end of get ip ----------

	// ---------- start log file ----------
	DateNow := StartTime.Format("2006-01-02")
	LogFILE := LogFile + "UserLogin_" + DateNow + ".log"
	file, err := os.OpenFile(LogFILE, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	log.SetOutput(file)
	// ---------- end of log file ----------

	// ------ start body json validation ------
	if c.Request.Body != nil {
		bodyBytes, _ = ioutil.ReadAll(c.Request.Body)
	}
	c.Request.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))
	bodyString := string(bodyBytes)

	bodyJson := helper.TrimReplace(string(bodyString))
	logData := StartTimeStr + "~" + IP + "~" + Method + "~" + Path + "~" + AllHeader + "~"
	rex := regexp.MustCompile(`\r?\n`)
	logData = logData + rex.ReplaceAllString(bodyJson, "") + "~"

	if string(bodyString) == "" {
		errorMessage := "Error, Body is empty"
		returnDataJsonUserLogin(jUserLoginResponses, totalPage, "1", "1", errorMessage, errorMessage, logData, c)
		helper.SendLogError(jUserLoginRequest.UsernameSession, PageGo, errorMessage, "", "", "1", AllHeader, Method, Path, IP, c)
		return
	}

	IsJson := helper.IsJson(bodyString)
	if !IsJson {
		errorMessage := "Error, Body - invalid json data"
		returnDataJsonUserLogin(jUserLoginResponses, totalPage, "1", "1", errorMessage, errorMessage, logData, c)
		helper.SendLogError(jUserLoginRequest.UsernameSession, PageGo, errorMessage, "", "", "1", AllHeader, Method, Path, IP, c)
		return
	}

	if helper.ValidateHeader(bodyString, c) {
		if err := c.ShouldBindJSON(&jUserLoginRequest); err != nil {
			errorMessage := "Error, Bind Json Data"
			returnDataJsonUserLogin(jUserLoginResponses, totalPage, "1", "1", errorMessage, errorMessage, logData, c)
			helper.SendLogError(jUserLoginRequest.UsernameSession, PageGo, errorMessage, "", "", "1", AllHeader, Method, Path, IP, c)
			return
		} else {
			Page := 0
			RowPage := 0

			UsernameMaster := jUserLoginRequest.UsernameMaster
			UsernameSession := jUserLoginRequest.UsernameSession
			Method := jUserLoginRequest.Method
			Id := jUserLoginRequest.Id
			Password := jUserLoginRequest.Password
			Nama := jUserLoginRequest.Nama
			Role := jUserLoginRequest.Role
			Status := jUserLoginRequest.Status
			ParamKeySession := jUserLoginRequest.ParamKey
			Page = jUserLoginRequest.Page
			RowPage = jUserLoginRequest.RowPage
			Order := jUserLoginRequest.Order
			OrderBy := jUserLoginRequest.OrderBy

			// ------ start check session paramkey ------
			checkAccessVal := helper.CheckSession(UsernameSession, ParamKeySession, c)
			if checkAccessVal != "1" {
				checkAccessValErrorMsg := checkAccessVal
				checkAccessValErrorMsgReturn := "Session Expired"
				returnDataJsonUserLogin(jUserLoginResponses, totalPage, "2", "2", checkAccessValErrorMsgReturn, checkAccessValErrorMsgReturn, logData, c)
				helper.SendLogError(UsernameSession, PageGo, checkAccessValErrorMsg, "", "", "2", AllHeader, Method, Path, IP, c)
				return
			}

			if Method == "INSERT" {
				errorMessage := "OK"

				if UsernameMaster == "" {
					errorMessage = "Username tidak boleh kosong!"
				} else if Password == "" {
					errorMessage = "Password tidak boleh kosong!"
				} else if Nama == "" {
					errorMessage = "Nama tidak boleh kosong!"
				} else if Role == "" {
					errorMessage = "Role tidak boleh kosong!"
				}

				if errorMessage == "OK" {
					Password := encodeText(Password)

					userExist := 0
					query := fmt.Sprintf("SELECT ifnull(count(1),0)cnt FROM siam_login WHERE status = 1 and username = '%s'", UsernameMaster)
					if err := db.QueryRow(query).Scan(&userExist); err != nil {
						errorMessage := fmt.Sprintf("Error running %q: %+v", query, err)
						returnDataJsonUserLogin(jUserLoginResponses, totalPage, "1", "1", errorMessage, errorMessage, logData, c)
						helper.SendLogError(UsernameSession, PageGo, errorMessage, "", "", "1", AllHeader, Method, Path, IP, c)
						return
					} else {
						if userExist > 0 {
							returnDataJsonUserLogin(jUserLoginResponses, totalPage, "1", "1", errorMessage, errorMessage, logData, c)
							helper.SendLogError(UsernameSession, PageGo, errorMessage, "", "", "1", AllHeader, Method, Path, IP, c)
							return
						}
					}

					query = fmt.Sprintf("INSERT into siam_login(username,password,nama, role, status)values('%s','%s','%s','%s',1)", UsernameMaster, Password, Nama, Role)
					if _, err = db.Exec(query); err != nil {
						errorMessage := fmt.Sprintf("Error running %q: %+v", query, err)
						returnDataJsonUserLogin(jUserLoginResponses, totalPage, "1", "1", errorMessage, errorMessage, logData, c)
						helper.SendLogError(UsernameSession, PageGo, errorMessage, "", "", "1", AllHeader, Method, Path, IP, c)
						return
					}

					currentTime := time.Now()
					currentTimeStr := currentTime.Format("01/02/2006 15:04:05")

					jUserLoginResponse.Username = UsernameMaster
					jUserLoginResponse.Password = Password
					jUserLoginResponse.Nama = Nama
					jUserLoginResponse.Role = Role
					jUserLoginResponse.Status = 1
					jUserLoginResponse.TanggalInput = currentTimeStr

					jUserLoginResponses = append(jUserLoginResponses, jUserLoginResponse)

					errorMessage := "Sukses insert data!"
					returnDataJsonUserLogin(jUserLoginResponses, totalPage, "0", "0", errorMessage, errorMessage, logData, c)

				} else {
					returnDataJsonUserLogin(jUserLoginResponses, totalPage, "1", "1", errorMessage, errorMessage, logData, c)
					helper.SendLogError(UsernameSession, PageGo, errorMessage, "", "", "1", AllHeader, Method, Path, IP, c)
					return
				}

			} else if Method == "UPDATE" {
				querySet := ""
				if Password != "" {
					Password := encodeText(Password)
					querySet += " password = '" + Password + "'"
				}

				if Nama != "" {
					if querySet != "" {
						querySet += " , "
					}
					querySet += " nama = '" + Nama + "'"
				}

				if Role != "" {
					if querySet != "" {
						querySet += " , "
					}
					querySet += " role = '" + Role + "'"
				}

				if Status != "" {
					if querySet != "" {
						querySet += " , "
					}

					iStatus, err := strconv.Atoi(Status)
					if err == nil {
						querySet += fmt.Sprintf(" status = %d ", iStatus)
					} else {
						errorMessage := "Error convert variable, " + err.Error()
						returnDataJsonUserLogin(jUserLoginResponses, totalPage, "1", "1", errorMessage, errorMessage, logData, c)
						helper.SendLogError(UsernameSession, PageGo, errorMessage, "", "", "1", AllHeader, Method, Path, IP, c)
						return
					}
				}

				query1 := fmt.Sprintf(`update siam_login set %s where id = %d ;`, querySet, Id)
				rows, err := db.Query(query1)
				defer rows.Close()
				if err != nil {
					errorMessage := "Error query, " + err.Error()
					returnDataJsonUserLogin(jUserLoginResponses, totalPage, "1", "1", errorMessage, errorMessage, logData, c)
					helper.SendLogError(UsernameSession, PageGo, errorMessage, "", "", "1", AllHeader, Method, Path, IP, c)
					return
				}

				query1 = fmt.Sprintf(`select id, username, password, nama, role, status, tgl_input from siam_login sl where id = %d`, Id)
				rows, err = db.Query(query1)
				defer rows.Close()
				if err != nil {
					errorMessage := "Error query, " + err.Error()
					returnDataJsonUserLogin(jUserLoginResponses, totalPage, "1", "1", errorMessage, errorMessage, logData, c)
					helper.SendLogError(UsernameSession, PageGo, errorMessage, "", "", "1", AllHeader, Method, Path, IP, c)
					return
				}

				for rows.Next() {
					err = rows.Scan(
						&jUserLoginResponse.Id,
						&jUserLoginResponse.Username,
						&jUserLoginResponse.Password,
						&jUserLoginResponse.Nama,
						&jUserLoginResponse.Role,
						&jUserLoginResponse.Status,
						&jUserLoginResponse.TanggalInput,
					)
				}

				jUserLoginResponses = append(jUserLoginResponses, jUserLoginResponse)

				errorMessage := "Sukses update data!"
				returnDataJsonUserLogin(jUserLoginResponses, totalPage, "0", "0", errorMessage, errorMessage, logData, c)

			} else if Method == "DELETE" {

				if Id > 0 {
					query1 := fmt.Sprintf(`update siam_login set status = 0 where id = %d ;`, Id)
					rows, err := db.Query(query1)
					defer rows.Close()
					if err != nil {
						errorMessage := "Error query, " + err.Error()
						returnDataJsonUserLogin(jUserLoginResponses, totalPage, "1", "1", errorMessage, errorMessage, logData, c)
						helper.SendLogError(UsernameSession, PageGo, errorMessage, "", "", "1", AllHeader, Method, Path, IP, c)
						return
					}
				} else {
					errorMessage := "Data tidak ditemukan!"
					returnDataJsonUserLogin(jUserLoginResponses, totalPage, "1", "1", errorMessage, errorMessage, logData, c)
					helper.SendLogError(UsernameSession, PageGo, errorMessage, "", "", "1", AllHeader, Method, Path, IP, c)
					return
				}

				errorMessage := "Sukses delete data!"
				returnDataJsonUserLogin(jUserLoginResponses, totalPage, "0", "0", errorMessage, errorMessage, logData, c)

			} else if Method == "SELECT" {

				PageNow := (Page - 1) * RowPage

				queryWhere := ""
				if Id > 0 {
					if queryWhere != "" {
						queryWhere += " AND "
					}

					queryWhere += fmt.Sprintf(" id = %d ", Id)
				}

				if UsernameMaster != "" {
					if queryWhere != "" {
						queryWhere += " AND "
					}

					queryWhere += " username LIKE '%" + UsernameMaster + "%' "
				}

				if Nama != "" {
					if queryWhere != "" {
						queryWhere += " AND "
					}

					queryWhere += " nama LIKE '%" + Nama + "%' "
				}

				if Role != "" {
					if queryWhere != "" {
						queryWhere += " AND "
					}

					queryWhere += " role LIKE '%" + Role + "%' "
				}

				if Status != "" {
					if queryWhere != "" {
						queryWhere += " AND "
					}

					iStatus, err := strconv.Atoi(Status)
					if err == nil {
						queryWhere += fmt.Sprintf(" status = %d ", iStatus)
					} else {
						errorMessage := "Error convert variable, " + err.Error()
						returnDataJsonUserLogin(jUserLoginResponses, totalPage, "1", "1", errorMessage, errorMessage, logData, c)
						helper.SendLogError(UsernameSession, PageGo, errorMessage, "", "", "1", AllHeader, Method, Path, IP, c)
						return
					}
				}

				if queryWhere != "" {
					queryWhere = " WHERE " + queryWhere
				}

				queryOrder := ""

				if OrderBy != "" {
					queryOrder = " ORDER BY " + OrderBy + " " + Order
				}

				totalRecords = 0
				totalPage = 0

				query := fmt.Sprintf("SELECT COUNT(1) AS cnt FROM siam_login %s ;", queryWhere)
				if err := db.QueryRow(query).Scan(&totalRecords); err != nil {
					errorMessage := "Error query, " + err.Error()
					returnDataJsonUserLogin(jUserLoginResponses, totalPage, "1", "1", errorMessage, errorMessage, logData, c)
					helper.SendLogError(UsernameSession, PageGo, errorMessage, "", "", "1", AllHeader, Method, Path, IP, c)
					return
				}
				totalPage = math.Ceil(float64(totalRecords) / float64(RowPage))

				query1 := fmt.Sprintf(`select id, username ,password ,nama ,role ,status, tgl_input from siam_login sl %s %s LIMIT %d,%d;`, queryWhere, queryOrder, PageNow, RowPage)
				rows, err := db.Query(query1)
				defer rows.Close()
				if err != nil {
					errorMessage := "Error query, " + err.Error()
					returnDataJsonUserLogin(jUserLoginResponses, totalPage, "1", "1", errorMessage, errorMessage, logData, c)
					helper.SendLogError(UsernameSession, PageGo, errorMessage, "", "", "1", AllHeader, Method, Path, IP, c)
					return
				}

				for rows.Next() {
					err = rows.Scan(
						&jUserLoginResponse.Id,
						&jUserLoginResponse.Username,
						&jUserLoginResponse.Password,
						&jUserLoginResponse.Nama,
						&jUserLoginResponse.Role,
						&jUserLoginResponse.Status,
						&jUserLoginResponse.TanggalInput,
					)

					jUserLoginResponses = append(jUserLoginResponses, jUserLoginResponse)

					if err != nil {
						errorMessage := fmt.Sprintf("Error running %q: %+v", query1, err)
						returnDataJsonUserLogin(jUserLoginResponses, totalPage, "1", "1", errorMessage, errorMessage, logData, c)
						helper.SendLogError(UsernameSession, PageGo, errorMessage, "", "", "1", AllHeader, Method, Path, IP, c)
						return
					}
				}

				errorMessage := "OK"
				returnDataJsonUserLogin(jUserLoginResponses, totalPage, "0", "0", errorMessage, errorMessage, logData, c)
				return

			} else {
				errorMessage := "Method not found"
				returnDataJsonUserLogin(jUserLoginResponses, totalPage, "1", "1", errorMessage, errorMessage, logData, c)
				helper.SendLogError(UsernameSession, PageGo, errorMessage, "", "", "1", AllHeader, Method, Path, IP, c)
				return
			}

		}
	}

}

func encodeText(text string) string {
	base64String := base64.StdEncoding.EncodeToString([]byte(text))
	SignatureKey := fmt.Sprintf("%x", md5.Sum([]byte(base64String)))
	return SignatureKey
}

func returnDataJsonUserLogin(jUserLoginResponse []JUserLoginResponse, TotalPage float64, ErrorCode string, ErrorCodeReturn string, ErrorMessage string, ErrorMessageReturn string, logData string, c *gin.Context) {
	if strings.Contains(ErrorMessage, "Error running") {
		ErrorMessage = "Error Execute data"
	}

	if ErrorCode == "504" {
		c.String(http.StatusUnauthorized, "")
	} else {
		currentTime := time.Now()
		currentTime1 := currentTime.Format("01/02/2006 15:04:05")

		c.PureJSON(http.StatusOK, gin.H{
			"ErrCode":    ErrorCode,
			"ErrMessage": ErrorMessage,
			"DateTime":   currentTime1,
			"Result":     jUserLoginResponse,
			"TotalPage":  TotalPage,
		})
	}

	startTime := time.Now()

	rex := regexp.MustCompile(`\r?\n`)
	endTime := time.Now()
	codeError := "200"

	diff := endTime.Sub(startTime)

	logDataNew := rex.ReplaceAllString(logData+codeError+"~"+endTime.String()+"~"+diff.String()+"~"+ErrorMessage, "")
	log.Println(logDataNew)

	runtime.GC()

	return
}
