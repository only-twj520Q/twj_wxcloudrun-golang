package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"wxcloudrun-golang/db/dao"
	"wxcloudrun-golang/db/model"

	"gorm.io/gorm"
)

// JsonResult 返回结构
type JsonResult struct {
	Code     int         `json:"code"`
	ErrorMsg string      `json:"errorMsg,omitempty"`
	Data     interface{} `json:"data"`
}

// IndexHandler 计数器接口
func IndexHandler(w http.ResponseWriter, r *http.Request) {
	data, err := getIndex()
	if err != nil {
		fmt.Fprint(w, "内部错误")
		return
	}
	fmt.Fprint(w, data)
}

// CounterHandler 计数器接口
func CounterHandler(w http.ResponseWriter, r *http.Request) {
	res := &JsonResult{}

	if r.Method == http.MethodGet {
		counter, err := getCurrentCounter()
		if err != nil {
			res.Code = -1
			res.ErrorMsg = err.Error()
		} else {
			res.Data = counter.Count
		}
	} else if r.Method == http.MethodPost {
		count, err := modifyCounter(r)
		if err != nil {
			res.Code = -1
			res.ErrorMsg = err.Error()
		} else {
			res.Data = count
		}
	} else {
		res.Code = -1
		res.ErrorMsg = fmt.Sprintf("请求方法 %s 不支持", r.Method)
	}

	msg, err := json.Marshal(res)
	if err != nil {
		fmt.Fprint(w, "内部错误")
		return
	}
	w.Header().Set("content-type", "application/json")
	w.Write(msg)
}

// modifyCounter 更新计数，自增或者清零
func modifyCounter(r *http.Request) (int32, error) {
	action, err := getAction(r)
	if err != nil {
		return 0, err
	}

	var count int32
	if action == "inc" {
		count, err = upsertCounter(r)
		if err != nil {
			return 0, err
		}
	} else if action == "clear" {
		err = clearCounter()
		if err != nil {
			return 0, err
		}
		count = 0
	} else {
		err = fmt.Errorf("参数 action : %s 错误", action)
	}

	return count, err
}

// upsertCounter 更新或修改计数器
func upsertCounter(r *http.Request) (int32, error) {
	currentCounter, err := getCurrentCounter()
	var count int32
	createdAt := time.Now()
	if err != nil && err != gorm.ErrRecordNotFound {
		return 0, err
	} else if err == gorm.ErrRecordNotFound {
		count = 1
		createdAt = time.Now()
	} else {
		count = currentCounter.Count + 1
		createdAt = currentCounter.CreatedAt
	}

	counter := &model.CounterModel{
		Id:        1,
		Count:     count,
		CreatedAt: createdAt,
		UpdatedAt: time.Now(),
	}
	err = dao.Imp.UpsertCounter(counter)
	if err != nil {
		return 0, err
	}
	return counter.Count, nil
}

func clearCounter() error {
	return dao.Imp.ClearCounter(1)
}

// getCurrentCounter 查询当前计数器
func getCurrentCounter() (*model.CounterModel, error) {
	counter, err := dao.Imp.GetCounter(1)
	if err != nil {
		return nil, err
	}

	return counter, nil
}

// getAction 获取action
func getAction(r *http.Request) (string, error) {
	decoder := json.NewDecoder(r.Body)
	body := make(map[string]interface{})
	if err := decoder.Decode(&body); err != nil {
		return "", err
	}
	defer r.Body.Close()

	action, ok := body["action"]
	if !ok {
		return "", fmt.Errorf("缺少 action 参数")
	}

	return action.(string), nil
}

// getIndex 获取主页
func getIndex() (string, error) {
	b, err := ioutil.ReadFile("./index.html")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func TestHandler(w http.ResponseWriter, r *http.Request) {
	res := &JsonResult{}

	token, err := getAccessToken()
	var p1 = "default1"
	var p2 = "default2"

	if err == nil {
		p1, p2, _ = addDraft(token.AccessToken)
	}

	if err != nil {
		res.Data = map[string]string{
			"resp": err.Error(),
			"body": p1,
			"url":  p2,
		}
	} else {
		res.Data = map[string]string{
			"body": p1,
			"url":  p2,
		}
	}

	msg, err := json.Marshal(res)
	if err != nil {
		fmt.Fprint(w, "内部错误")
		return
	}

	w.Header().Set("content-type", "application/json")
	w.Write(msg)
}

type AccessTokenResp struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}

func getAccessToken() (AccessTokenResp, error) {
	client := &http.Client{}

	// 创建HTTP请求
	req, err := http.NewRequest("GET", "https://api.weixin.qq.com/cgi-bin/token?grant_type=client_credential", nil)
	if err != nil {
		panic(err)
	}

	q := req.URL.Query()
	q.Add("grant_type", "client_credential")
	q.Add("appid", "wxb3a518d1232e62d5")
	q.Add("secret", "46fdcdec4ac0f86a113eda2cc83470a2")

	req.URL.RawQuery = q.Encode()

	// 发送HTTP请求
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}

	defer resp.Body.Close() // 确保关闭响应体

	// 读取响应体
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	token := AccessTokenResp{}
	err = json.Unmarshal(body, &token)

	return token, err
}

func addDraft(token string) (string, string, error) {
	client := &http.Client{}

	var data = map[string][]interface{}{}
	var content = map[string]interface{}{
		"title":          "澳洲newest",
		"digest":         "前几个字",
		"content":        "礼物推荐-3",
		"thumb_media_id": "IUHZ5Ned3t6_I_bpWRQx_3aGQ3ryMNp3fP4GDEd4tIj2S37ANxQ2vPPCHLk7F1xc",
	}

	data["articles"] = append(data["articles"], content)

	bytesData, _ := json.Marshal(data)

	log.Printf("addDraft param=%v", string(bytesData))

	// 创建HTTP请求
	req, err := http.NewRequest("POST", "https://api.weixin.qq.com/cgi-bin/draft/add", bytes.NewReader(bytesData))
	if err != nil {
		panic(err)
	}

	q := req.URL.Query()
	q.Add("access_token", token)

	req.URL.RawQuery = q.Encode()

	// 发送HTTP请求
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}

	defer resp.Body.Close() // 确保关闭响应体

	// 读取响应体
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	log.Printf("addDraft body=%v", string(body))

	return string(bytesData), string(body), nil
}
