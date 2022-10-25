// hello world
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"
)

// 设置每次抢购的间隔时间
const SECKILL_TIME = 500 * time.Millisecond

const COUNT = 10

type User struct {
	NickName    string `json:"nickName"`
	Email       string `json:"email"`
	Password    string `json:"password"`
	PayPassword string `json:"pay_password"`
}

type LoginRes struct {
	AccessToken      string `json:"access_token"`
	ExpiresIn        int    `json:"expires_in"`
	RefreshToken     string `json:"refresh_token"`
	RefreshExpiresIn int    `json:"refresh_expires_in"`
	TokenType        string `json:"token_type"`
}

// 请求可选参数
type RequestOptions struct {
	token string
}

type Detail struct {
	Startup Startup  `json:"startup"`
	Issues  []Issues `json:"issues"`
}
type Title struct {
	Chinese string `json:"chinese"`
	English string `json:"english"`
}
type Describe struct {
	Chinese string `json:"chinese"`
	English string `json:"english"`
}
type Introduce struct {
	Chinese string `json:"chinese"`
	English string `json:"english"`
}
type IssuerName struct {
	Chinese string `json:"chinese"`
	English string `json:"english"`
}
type Startup struct {
	ID          string     `json:"id"`
	Startup     bool       `json:"startup"`
	Title       Title      `json:"title"`
	Image       string     `json:"image"`
	Issuer      string     `json:"issuer"`
	Describe    Describe   `json:"describe"`
	Introduce   Introduce  `json:"introduce"`
	Contract    string     `json:"contract"`
	Issuing     bool       `json:"issuing"`
	IssueIndex  string     `json:"issue_index"`
	Finished    bool       `json:"finished"`
	Mystery     bool       `json:"mystery"`
	SoldOut     bool       `json:"sold_out"`
	InWhitelist bool       `json:"in_whitelist"`
	Limit       bool       `json:"limit"`
	Total       string     `json:"total"`
	CanBuy      int        `json:"can_buy"`
	PureIndex   string     `json:"pure_index"`
	Price       string     `json:"price"`
	IssuerName  IssuerName `json:"issuer_name"`
	Time        string     `json:"time"`
}
type Issues struct {
	Seq         string `json:"seq"`
	Title       Title  `json:"title"`
	StartTime   string `json:"start_time"`
	EndTime     string `json:"end_time"`
	Currency    string `json:"currency"`
	Type        string `json:"type"`
	Limit       string `json:"limit"`
	TotalAmount string `json:"total_amount"`
	Price       string `json:"price"`
	CurrencyID  string `json:"currency_id"`
	Sold        string `json:"sold"`
}

const PROD_URL = ""

// 初始化用户
var users *[]User

// 构建一个用户和token映射的字典
var userTokenMap = make(map[string]string)

// 构建一个用户和藏品信息映射的字典
var userDetailMap = make(map[string]Detail)

func main() {
	u, err := parseUserJson()
	if err != nil {
		return
	} else {
		// 赋值给全局变量
		users = u
	}
	// 终端提示, 是否开始批量登陆
	fmt.Println("====是否开始批量登陆？(y/n)====")
	var input string
	fmt.Scanln(&input)
	if input == "y" {
		// 开始批量登陆
		login()
	} else {
		os.Exit(0)
	}
	// 输入抢购商品id
	fmt.Println("====请输入抢购商品id====")
	var productId int
	fmt.Scanln(&productId)
	detail(productId)
	// 是否开始抢购
	fmt.Println("====是否开始抢购？(y/n)====")
	fmt.Scanln(&input)
	if input == "y" {
		// 开始抢购
		seckill(productId)
	} else {
		os.Exit(0)
	}
}

// 查看详情
func detail(productId int) {
	var wg sync.WaitGroup
	for _, user := range *users {
		wg.Add(1)
		go func(user User) {
			defer wg.Done()
			// 通过userTokenMap获取token
			token := userTokenMap[user.Email]
			result, err := httpDo[Detail](fmt.Sprintf("%s/user/startups/%d", PROD_URL, productId), nil, "GET", RequestOptions{
				token: token,
			})
			if err != nil {
				fmt.Println("查询失败", err)
			} else {
				// 赋值给全局变量
				userDetailMap[user.Email] = *result
			}
		}(user)
	}
	wg.Wait()
}

// 解析user.json
func parseUserJson() (*[]User, error) {
	filePtr, err := os.Open("./user.json")
	if err != nil {
		fmt.Println("user文件打开失败 [Err:%s]", err.Error())
		return nil, err
	}
	defer filePtr.Close()
	var users []User
	decoder := json.NewDecoder(filePtr)
	err = decoder.Decode(&users)
	if err != nil {
		fmt.Println("解码用户失败", err.Error())
		return nil, err
	} else {
		fmt.Println("解码用户成功")
		return &users, nil
	}
}

// 抢购
func seckill(productId int) {
	// 所有用户并发抢购
	var wg sync.WaitGroup
	for email, token := range userTokenMap {
		wg.Add(1)
		go func(email string, token string) {
			defer wg.Done()
			isClose := make(chan struct{})
			// COUNT作为默认值
			count := COUNT
			// 获取支付密码, 循环users数组, 通过email匹配
			var currentUser *User
			for _, user := range *users {
				if user.Email == email {
					currentUser = &user
					break
				}
			}
			FOR: for {
				select {
				case <-isClose:
					break
				default:
					// 如果count为0, 则退出循环
					if count == 0 {
						fmt.Println("用户", email, "抢购结束, 共抢购", COUNT-count, "次")
						break FOR
					}
					go func() {
						_, err := httpDo[any](fmt.Sprintf("%s/startups/%d/mint", PROD_URL, productId), map[string]any{
							"amount:":     userDetailMap[email].Startup.Price,
							"currency_id": "1000000",
							"issue_index": userDetailMap[email].Startup.IssueIndex,
							"password":    currentUser.PayPassword,
						}, "POST", RequestOptions{
							token: token,
						})
						if err != nil {
							fmt.Println("用户", email, "抢购失败😈")
						} else {
							fmt.Println("用户", email, "抢购成功😄")
							isClose <- struct{}{}
						}
					}()
					count --;
					time.Sleep(SECKILL_TIME)
				}
			}
		}(email, token)
	}
	wg.Wait()
}

// 批量登陆
func login() {
	// 并发登陆所有用户
	var wg sync.WaitGroup
	for _, user := range *users {
		wg.Add(1)
		go func(user User) {
			defer wg.Done()
			result, err := httpDo[LoginRes]("/login", map[string]any{
				"password": user.Password,
				"username": user.Email,
			}, "POST")
			userTokenMap[user.Email] = result.AccessToken
			if err != nil {
				fmt.Println("用户", user.NickName, "登陆失败")
			} else {
				fmt.Println("用户", user.NickName, "登陆成功")
			}
		}(user)
	}
	wg.Wait()
	fmt.Println("所有用户登陆完成")
}

// 封装一个http请求
func httpDo[T any](url string, params map[string]any, method string, options ...RequestOptions) (*T, error) {
	content, err := json.Marshal(params)

	if err != nil {
		return nil, err
	}

	body := bytes.NewBuffer(content)

	request, err := http.NewRequest(method,
		url,
		body)
	// json
	request.Header.Set("Content-Type", "application/json")
	request.Header.Add("User-Agent",
		"Mozilla/5.0 (iPhone; CPU iPhone OS 11_0 like Mac OS X) AppleWebKit/604.1.38 (KHTML, like Gecko) Version/11.0 Mobile/15A372 Safari/604.1")

	// option如果不为空
	if len(options) > 0 {
		request.Header.Add("authorization", "Bearer "+options[0].token)
	}
	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// 判断状态码
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("请求失败，状态码：%d", resp.StatusCode)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result T
	err = json.Unmarshal(respBody, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}
