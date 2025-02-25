package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gclog"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gctemplates"
	"github.com/gochan-org/gochan/pkg/posting"
	"github.com/gochan-org/gochan/pkg/serverutil"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

var (
	versionStr string
	stdFatal   = gclog.LStdLog | gclog.LFatal
)

func main() {
	defer func() {
		gclog.Print(gclog.LStdLog, "Cleaning up")
		//gcsql.ExecSQL("DROP TABLE DBPREFIXsessions")
		gcsql.Close()
	}()

	gclog.Printf(gclog.LStdLog, "Starting gochan v%s", versionStr)
	config.InitConfig(versionStr)

	systemCritical := config.GetSystemCriticalConfig()

	gcsql.ConnectToDB(
		systemCritical.DBhost, systemCritical.DBtype, systemCritical.DBname,
		systemCritical.DBusername, systemCritical.DBpassword, systemCritical.DBprefix)
	gcsql.CheckAndInitializeDatabase(systemCritical.DBtype)
	parseCommandLine()
	serverutil.InitMinifier()

	posting.InitCaptcha()
	if err := gctemplates.InitTemplates(); err != nil {
		gclog.Printf(gclog.LErrorLog|gclog.LStdLog|gclog.LFatal, err.Error())
	}

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	posting.InitPosting()
	go initServer()
	<-sc
}

func parseCommandLine() {
	var newstaff string
	var delstaff string
	var rebuild string
	var rank int
	var err error
	flag.StringVar(&newstaff, "newstaff", "", "<newusername>:<newpassword>")
	flag.StringVar(&delstaff, "delstaff", "", "<username>")
	flag.StringVar(&rebuild, "rebuild", "", "accepted values are boards,front,js, or all")
	flag.IntVar(&rank, "rank", 0, "New staff member rank, to be used with -newstaff or -delstaff")
	flag.Parse()

	rebuildFlag := buildNone
	switch rebuild {
	case "boards":
		rebuildFlag = buildBoards
	case "front":
		rebuildFlag = buildFront
	case "js":
		rebuildFlag = buildJS
	case "all":
		rebuildFlag = buildAll
	}
	if rebuildFlag > 0 {
		startupRebuild(rebuildFlag)
	}

	if newstaff != "" {
		arr := strings.Split(newstaff, ":")
		if len(arr) < 2 || delstaff != "" {
			flag.Usage()
			os.Exit(1)
		}
		gclog.Printf(gclog.LStdLog|gclog.LStaffLog, "Creating new staff: %q, with password: %q and rank: %d from command line", arr[0], arr[1], rank)
		if err = gcsql.NewStaff(arr[0], arr[1], rank); err != nil {
			gclog.Print(stdFatal, err.Error())
		}
		os.Exit(0)
	}
	if delstaff != "" {
		if newstaff != "" {
			flag.Usage()
			os.Exit(1)
		}
		gclog.Printf(gclog.LStdLog, "Are you sure you want to delete the staff account %q? [y/N]: ", delstaff)
		var answer string
		fmt.Scanln(&answer)
		answer = strings.ToLower(answer)
		if answer == "y" || answer == "yes" {
			if err = gcsql.DeleteStaff(delstaff); err != nil {
				gclog.Printf(stdFatal, "Error deleting %q: %s", delstaff, err.Error())
			}
		} else {
			gclog.Print(stdFatal, "Not deleting.")
		}
	}
}
