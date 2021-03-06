package src

import (
	"fmt"
	"github.com/fatih/color"
	"github.com/urfave/cli"
	"io/ioutil"
	"math"
	"os"
	"regexp"
	"strconv"
	"strings"
)

func CreateCmdAction(ctx *cli.Context) {
	var content, keyword, comment string
	args := ctx.Args()

	switch len(args) {
	case 0:
		color.Red("please input content at least")
		return
	case 1:
		content = args[0]
	case 2:
		content, keyword = args[0], args[1]
	case 3:
		content, keyword, comment = args[0], args[1], args[2]
	default:
		break
	}

	if len(keyword) > 0 {
		if ok, _ := regexp.MatchString("^[a-zA-Z]{1}([a-zA-Z0-9]|[._\\-]){2,254}$", keyword); !ok {
			fmt.Println("Keyword can only contains 3-255 strings starting with a letter, with numbers, '_', '.','-', and must start with a letter")
			return
		}
	}

	cmd := Cmd{
		Cmd:     content,
		Comment: comment,
		Keyword: keyword,
		Private: !ctx.Bool("public"),
	}
	result := Create(cmd)
	fmt.Printf("content: %s\n", content)
	fmt.Printf("keyword: %s\n", keyword)
	fmt.Printf("comment: %s\n", comment)
	printRespond(result)
}

func ForkCmdAction(ctx *cli.Context) {
	var keyword string
	args := ctx.Args()

	if len(args) == 0 {
		color.Red("please input keyword")
		return
	} else {
		keyword = args[0]
	}

	result := ForkCmd(keyword)
	printRespond(result)
}

// cmd
func SearchAction(ctx *cli.Context) {
	page := ctx.Int("page")
	size := ctx.Int("size")
	var result CmdsRespond
	if ctx.Bool("all") || len(ctx.Args()) == 0 {
		result = Search("", page, size)
	} else {
		result = Search(ctx.Args()[0], page, size)
	}

	green := color.New(color.FgGreen).SprintFunc()
	blue := color.New(color.FgBlue).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()
	magenta := color.New(color.FgMagenta).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()

	cmds := result.Data
	length := len(cmds)
	for i := 0; i < length; i++ {
		status := "public"
		if cmds[i].Private {
			status = magenta("private")
		}
		fmt.Fprintf(color.Output, "%s | %s %s %s id:%s\n", green(cmds[i].Cmd), blue(cmds[i].Keyword), cyan(cmds[i].Comment), status, red(cmds[i].Id))
	}
	if result.Status == SUCCESS {
		total, _ := strconv.Atoi(result.Message)
		var totalPage float64 = 0
		if total != 0 {
			totalPage = math.Ceil(float64(total) / float64(size))
		}
		fmt.Fprintf(color.Output, "total:%v, size:%v, page:%v/%v\n", total, size, page, totalPage)
	} else {
		color.Red(result.Message)
	}
}

func insertParams(content string, args []string) string {
	if len(args) == 1 {
		return content
	}
	args = args[1:]
	// 优先把具名变量填入，再依次填入匿名参数
	var nameVar []string
	var anonymousVar []string
	for i := 0; i < len(args); i++ {
		if strings.Contains(args[i], "=") {
			nameVar = append(nameVar, args[i])
		} else {
			anonymousVar = append(anonymousVar, args[i])
		}
	}
	// 先替换具名参数
	var nameAndValue []string
	for i := 0; i < len(nameVar); i++ {
		nameAndValue = strings.Split(nameVar[i], "=")
		reg := regexp.MustCompile(`{{\s*` + nameAndValue[0] + `\s*}}`)
		content = reg.ReplaceAllString(content, nameAndValue[1])
	}

	// 再替换匿名参数
	reg := regexp.MustCompile(`{{\s*[[:alnum:]]*\s*}}`)
	results := reg.FindAllString(content, -1)
	for i := 0; i < len(anonymousVar); i++ {
		content = strings.Replace(content, results[i], anonymousVar[i], 1)
	}
	return content
}

func ExecAction(ctx *cli.Context) {
	keyword := ctx.Args()[0]
	if len(keyword) == 0 {
		color.Red("please input keyword")
		return
	}

	if ctx.Bool("file") {
		result := DownloadFile(ctx.Args()[0])
		if result.Status == SUCCESS {
			file := result.Data
			if ctx.Bool("print") || (strings.Contains(keyword, "/") && !ctx.Bool("force")) {
				fmt.Println(file.Content)
				return
			}
			// 在执行之前，替换占位参数
			result.Data.Content = insertParams(result.Data.Content, ctx.Args())
			output, err := Exec(result.Data.Content)
			if err != nil {
				color.Red("exec fail")
			} else {
				if len(output) > 0 {
					fmt.Print(output)
				}
				//color.Green("success")
			}
		} else {
			color.Red(result.Message)
		}
	} else {
		result := GetCmd(ctx.Args()[0])
		if result.Status == SUCCESS {
			cmd := result.Data
			// 如果keyword中包含/，那么就是引用他人的命令，默认是显示不执行，如果加了--force或-F的话，才可以执行。
			// 仅防他人的危险命令！
			// 自己的命令可以直接执行
			if ctx.Bool("print") || (strings.Contains(keyword, "/") && !ctx.Bool("force")) {
				fmt.Println(cmd.Cmd)
				return
			}
			// 在执行之前，替换占位参数
			result.Data.Cmd = insertParams(result.Data.Cmd, ctx.Args())
			output, err := Exec(result.Data.Cmd)
			if err != nil {
				color.Red("error")
			} else {
				if len(output) > 0 {
					fmt.Print(output)
				}
				//color.Green(result.Message)
			}
		} else {
			color.Red(result.Message)
		}
	}
}

func DeleteCmdAction(ctx *cli.Context) {
	args := ctx.Args()
	if len(args) == 0 {
		color.Red("please input id")
		return
	}
	result := Delete(args[0])
	printRespond(result)
}

// user

func RegisterAction(ctx *cli.Context) {
	args := ctx.Args()
	if len(args) < 2 {
		color.Red("please input username and password")
		return
	}

	if ok, _ := regexp.MatchString("^[a-zA-Z]{1}([a-zA-Z0-9]|[._\\-]){2,19}$", args[0]); !ok {
		fmt.Println("You can only enter 3-20 strings starting with a letter, with numbers, '_', '.','-', and must start with a letter")
		return
	}

	result := Register(args[0], args[1])
	// 把token写入本地
	if result.Status != SUCCESS {
		printRespond(result)
		return
	}
	CreateToken(result.Data)
	result.Message = "success"
	printRespond(result)
}

func LoginAction(ctx *cli.Context) {
	args := ctx.Args()
	if len(args) < 2 {
		color.Red("please input username and password")
		return
	}
	result := Login(args[0], args[1])
	// 把token写入本地
	if result.Status != SUCCESS {
		printRespond(result)
		return
	}
	CreateToken(result.Data)
	result.Message = "success"
	printRespond(result)
}

func LogoutAction(ctx *cli.Context) {
	CreateToken("")
	color.Green("success")
}

func ResetPasswordAction(ctx *cli.Context) {
	result := ResetPassword(ctx.Args()[0])
	// 把token写入本地
	if result.Status != SUCCESS {
		printRespond(result)
		return
	}
	CreateToken(result.Data)
	printRespond(result)
}

func UpdateInfoAction(ctx *cli.Context) {
	result := UpdateInfo(ctx.Args()[0])
	// 把token写入本地
	printRespond(result)
}

// file

func PushFileAction(ctx *cli.Context) {
	var filePath, keyword, comment string
	args := ctx.Args()

	switch len(args) {
	case 0:
		color.Red("please input file path and keyword at least")
		return
	case 1:
		filePath = args[0]
	case 2:
		filePath, keyword = args[0], args[1]
	case 3:
		filePath, keyword, comment = args[0], args[1], args[2]
	default:
		break
	}

	fileData, err := os.Open(filePath)
	if err != nil {
		fmt.Println(err)
		return
	}
	fileContent, err := ioutil.ReadFile(filePath)
	if err != nil {
		fmt.Println(err)
		return
	}
	// keyword默认等于文件名
	if len(args) == 1 {
		keyword = fileData.Name()
	}

	fileNameSplit := strings.Split(fileData.Name(), "/")
	fileName := fileNameSplit[len(fileNameSplit)-1]

	file := File{
		Name:    fileName,
		Content: string(fileContent),
		Comment: comment,
		Keyword: keyword,
		Private: !ctx.Bool("public"),
	}

	fileData.Close()
	result := CreateFile(&file)
	fmt.Printf("file path: %s\n", filePath)
	fmt.Printf("keyword: %s\n", keyword)
	fmt.Printf("comment: %s\n", comment)
	printRespond(result)
}

func ForkFileAction(ctx *cli.Context) {
	var keyword string
	args := ctx.Args()

	if len(args) == 0 {
		color.Red("please input keyword")
		return
	} else {
		keyword = args[0]
	}

	result := ForkFile(keyword)
	printRespond(result)
}

func PullFileAction(ctx *cli.Context) {
	result := DownloadFile(ctx.Args()[0])

	green := color.New(color.FgGreen).SprintFunc()
	blue := color.New(color.FgBlue).SprintFunc()

	red := color.New(color.FgRed).SprintFunc()

	if result.Status == SUCCESS {
		file := result.Data
		status := "public"
		if file.Private {
			status = "private"
		}
		// 在生成文件之前，替换占位参数
		file.Content = insertParams(file.Content, ctx.Args())

		if ctx.Bool("print") {
			fmt.Println(file.Content)
			return
		} else {
			err := ioutil.WriteFile(file.Name, []byte(file.Content), 0644)
			if err != nil {
				fmt.Println(err)
				return
			}
			fmt.Fprintf(color.Output, "%s | %s %s %s id:%s\n", green(file.Name), blue(file.Keyword), file.Comment, status, red(file.Id))
			fmt.Println(file.Content)
		}
		color.Green(result.Message)
	} else {
		color.Red(result.Message)
	}
}

func FindFileAction(ctx *cli.Context) {
	page := ctx.Int("page")
	size := ctx.Int("size")
	var result FilesRespond
	if ctx.Bool("all") || len(ctx.Args()) == 0 {
		result = SearchFile("", page, size)
	} else {
		result = SearchFile(ctx.Args()[0], page, size)
	}

	green := color.New(color.FgGreen).SprintFunc()
	blue := color.New(color.FgBlue).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()
	magenta := color.New(color.FgMagenta).SprintFunc()

	files := result.Data
	length := len(files)
	for i := 0; i < length; i++ {
		status := "public"
		if files[i].Private {
			status = magenta("private")
		}
		fmt.Fprintf(color.Output, "%s | %s %s %s id:%s\n", green(files[i].Name), blue(files[i].Keyword), cyan(files[i].Comment), status, red(files[i].Id))
	}
	if result.Status == SUCCESS {
		total, _ := strconv.Atoi(result.Message)
		var totalPage float64 = 0
		if total != 0 {
			totalPage = math.Ceil(float64(total) / float64(size))
		}
		fmt.Fprintf(color.Output, "total:%v, size:%v, page:%v/%v\n", total, size, page, totalPage)
	} else {
		color.Red(result.Message)
	}
}

func RemoveFileAction(ctx *cli.Context) {
	args := ctx.Args()
	if len(args) == 0 {
		color.Red("please input id")
		return
	}
	result := DeleteFile(args[0])
	printRespond(result)
}

// star

func StarAction(ctx *cli.Context) {
	if ctx.Int("delete") > 0 {
		result := DeleteStar(ctx.Int("delete"))
		if result.Status == SUCCESS {
			color.Green(result.Message)
		} else {
			color.Red(result.Message)
		}
	} else {
		if len(ctx.Args()) == 0 {
			page := ctx.Int("page")
			size := ctx.Int("size")
			keyword := ctx.String("keyword")
			result := SearchStar(page, size, keyword)

			green := color.New(color.FgGreen).SprintFunc()
			blue := color.New(color.FgBlue).SprintFunc()
			cyan := color.New(color.FgCyan).SprintFunc()
			red := color.New(color.FgRed).SprintFunc()
			magenta := color.New(color.FgMagenta).SprintFunc()

			stars := result.Data
			length := len(stars)
			for i := 0; i < length; i++ {
				fmt.Fprintf(color.Output, "%s | %s %s | star: %v, cmds: %v, files: %v | id:%s\n", green(stars[i].User.Username), blue(stars[i].User.Info), cyan(stars[i].User.CreatedAt.Format("2006-01")), cyan(stars[i].StarCount), red(stars[i].CmdCount), magenta(stars[i].FileCount), red(stars[i].StarId))
			}
			if result.Status == SUCCESS {
				total, _ := strconv.Atoi(result.Message)
				var totalPage float64 = 0
				if total != 0 {
					totalPage = math.Ceil(float64(total) / float64(size))
				}
				fmt.Fprintf(color.Output, "total:%v, size:%v, page:%v/%v\n", total, size, page, totalPage)
			} else {
				color.Red(result.Message)
			}
		} else {
			result := CreateStar(ctx.Args()[0])
			printRespond(result)
		}
	}
}

func UpdateAction(ctx *cli.Context) {
	_, err := Exec("go get github.com/yurencloud/cmdp")
	if err != nil {
		fmt.Println(err)
		return
	}
}

func UserAction(ctx *cli.Context) {
	page := ctx.Int("page")
	size := ctx.Int("size")
	var official = ""

	if ctx.Bool("official") {
		official = "1"
	} else {
		official = "0"
	}
	var result UsersRespond
	if ctx.Bool("all") || len(ctx.Args()) == 0 {
		result = SearchUser(page, size, "", official)
	} else {
		result = SearchUser(page, size, ctx.Args()[0], official)
	}

	green := color.New(color.FgGreen).SprintFunc()
	blue := color.New(color.FgBlue).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()
	magenta := color.New(color.FgMagenta).SprintFunc()

	users := result.Data
	length := len(users)
	for i := 0; i < length; i++ {
		fmt.Fprintf(color.Output, "%-24s | star: %v, cmds: %v, files: %v %s\n", green(users[i].Username), cyan(users[i].StarCount), red(users[i].CmdCount), magenta(users[i].FileCount), blue(users[i].Info))
	}
	if result.Status == SUCCESS {
		total, _ := strconv.Atoi(result.Message)
		var totalPage float64 = 0
		if total != 0 {
			totalPage = math.Ceil(float64(total) / float64(size))
		}
		fmt.Fprintf(color.Output, "total:%v, size:%v, page:%v/%v\n", total, size, page, totalPage)
	} else {
		color.Red(result.Message)
	}
}
