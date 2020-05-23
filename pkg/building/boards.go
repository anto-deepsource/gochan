package building

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"strconv"
	"syscall"
	"time"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gclog"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gctemplates"
	"github.com/gochan-org/gochan/pkg/gcutil"
)

const (
	dirIsAFileStr = `unable to create "%s", path exists and is a file`
	genericErrStr = `unable to create "%s": %s`
	pathExistsStr = `unable to create "%s", path already exists`
)

// BuildBoardPages builds the pages for the board archive.
// `board` is a Board object representing the board to build archive pages for.
// The return value is a string of HTML with debug information from the build process.
func BuildBoardPages(board *gcsql.Board) (html string) {
	err := gctemplates.InitTemplates("boardpage")
	if err != nil {
		return err.Error()
	}
	var currentPageFile *os.File
	var threads []interface{}
	var threadPages [][]interface{}
	var stickiedThreads []interface{}
	var nonStickiedThreads []interface{}
	var opPosts []gcsql.Post

	// Get all top level posts for the board.
	if opPosts, err = gcsql.GetTopPosts(board.ID, true); err != nil {
		return html + gclog.Printf(gclog.LErrorLog,
			"Error getting OP posts for /%s/: %s", board.Dir, err.Error()) + "<br />"
	}

	// For each top level post, start building a Thread struct
	for _, op := range opPosts {
		var thread gcsql.Thread
		var postsInThread []gcsql.Post

		var replyCount, err = gcsql.GetReplyCount(op.ID)
		if err != nil {
			return html + gclog.Printf(gclog.LErrorLog,
				"Error getting replies to /%s/%d: %s",
				board.Dir, op.ID, err.Error()) + "<br />"
		}
		thread.NumReplies = replyCount

		fileCount, err := gcsql.GetReplyFileCount(op.ID)
		if err != nil {
			return html + gclog.Printf(gclog.LErrorLog,
				"Error getting file count to /%s/%d: %s",
				board.Dir, op.ID, err.Error()) + "<br />"
		}
		thread.NumImages = fileCount

		thread.OP = op

		var numRepliesOnBoardPage int

		if op.Stickied {
			// If the thread is stickied, limit replies on the archive page to the
			// configured value for stickied threads.
			numRepliesOnBoardPage = config.Config.StickyRepliesOnBoardPage
		} else {
			// Otherwise, limit the replies to the configured value for normal threads.
			numRepliesOnBoardPage = config.Config.RepliesOnBoardPage
		}

		postsInThread, err = gcsql.GetExistingRepliesLimitedRev(op.ID, numRepliesOnBoardPage)
		if err != nil {
			return html + gclog.Printf(gclog.LErrorLog,
				"Error getting posts in /%s/%d: %s",
				board.Dir, op.ID, err.Error()) + "<br />"
		}

		var reversedPosts []gcsql.Post
		for i := len(postsInThread); i > 0; i-- {
			reversedPosts = append(reversedPosts, postsInThread[i-1])
		}

		if len(postsInThread) > 0 {
			// Store the posts to show on board page
			//thread.BoardReplies = postsInThread
			thread.BoardReplies = reversedPosts

			// Count number of images on board page
			imageCount := 0
			for _, reply := range postsInThread {
				if reply.Filesize != 0 {
					imageCount++
				}
			}
			// Then calculate number of omitted images.
			thread.OmittedImages = thread.NumImages - imageCount
		}

		// Add thread struct to appropriate list
		if op.Stickied {
			stickiedThreads = append(stickiedThreads, thread)
		} else {
			nonStickiedThreads = append(nonStickiedThreads, thread)
		}
	}

	gcutil.DeleteMatchingFiles(path.Join(config.Config.DocumentRoot, board.Dir), "\\d.html$")
	// Order the threads, stickied threads first, then nonstickied threads.
	threads = append(stickiedThreads, nonStickiedThreads...)

	// If there are no posts on the board
	if len(threads) == 0 {
		board.CurrentPage = 1
		// Open board.html for writing to the first page.
		boardPageFile, err := os.OpenFile(path.Join(config.Config.DocumentRoot, board.Dir, "board.html"), os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0777)
		if err != nil {
			return html + gclog.Printf(gclog.LErrorLog,
				"Failed opening /%s/board.html: %s",
				board.Dir, err.Error()) + "<br />"
		}

		// Render board page template to the file,
		// packaging the board/section list, threads, and board info
		if err = gcutil.MinifyTemplate(gctemplates.BoardPage, map[string]interface{}{
			"config":   config.Config,
			"boards":   gcsql.AllBoards,
			"sections": gcsql.AllSections,
			"threads":  threads,
			"board":    board,
		}, boardPageFile, "text/html"); err != nil {
			return html + gclog.Printf(gclog.LErrorLog,
				"Failed building /%s/: %s",
				board.Dir, err.Error()) + "<br />"
		}
		html += "/" + board.Dir + "/ built successfully.\n"
		return
	}

	// Create the archive pages.
	threadPages = paginate(config.Config.ThreadsPerPage, threads)
	board.NumPages = len(threadPages)

	// Create array of page wrapper objects, and open the file.
	pagesArr := make([]map[string]interface{}, board.NumPages)

	catalogJSONFile, err := os.OpenFile(path.Join(config.Config.DocumentRoot, board.Dir, "catalog.json"), os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0777)
	if err != nil {
		return gclog.Printf(gclog.LErrorLog,
			"Failed opening /%s/catalog.json: %s", board.Dir, err.Error()) + "<br />"
	}
	defer catalogJSONFile.Close()

	currentBoardPage := board.CurrentPage
	for _, pageThreads := range threadPages {
		board.CurrentPage++
		var currentPageFilepath string
		pageFilename := strconv.Itoa(board.CurrentPage) + ".html"
		currentPageFilepath = path.Join(config.Config.DocumentRoot, board.Dir, pageFilename)
		currentPageFile, err = os.OpenFile(currentPageFilepath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0777)
		if err != nil {
			html += gclog.Printf(gclog.LErrorLog, "Failed opening /%s/%s: %s",
				board.Dir, pageFilename, err.Error()) + "<br />"
			continue
		}
		defer currentPageFile.Close()

		// Render the boardpage template
		if err = gcutil.MinifyTemplate(gctemplates.BoardPage, map[string]interface{}{
			"config":   config.Config,
			"boards":   gcsql.AllBoards,
			"sections": gcsql.AllSections,
			"threads":  pageThreads,
			"board":    board,
			"posts": []interface{}{
				gcsql.Post{BoardID: board.ID},
			},
		}, currentPageFile, "text/html"); err != nil {
			return html + gclog.Printf(gclog.LErrorLog,
				"Failed building /%s/ boardpage: %s", board.Dir, err.Error()) + "<br />"
		}

		if board.CurrentPage == 1 {
			boardPage := path.Join(config.Config.DocumentRoot, board.Dir, "board.html")
			os.Remove(boardPage)
			if err = syscall.Symlink(currentPageFilepath, boardPage); !os.IsExist(err) && err != nil {
				html += gclog.Printf(gclog.LErrorLog, "Failed building /%s/: %s",
					board.Dir, err.Error())
			}
		}

		// Collect up threads for this page.
		pageMap := make(map[string]interface{})
		pageMap["page"] = board.CurrentPage
		pageMap["threads"] = pageThreads
		pagesArr = append(pagesArr, pageMap)
	}
	board.CurrentPage = currentBoardPage

	catalogJSON, err := json.Marshal(pagesArr)
	if err != nil {
		return html + gclog.Print(gclog.LErrorLog, "Failed to marshal to JSON: ", err.Error()) + "<br />"
	}
	if _, err = catalogJSONFile.Write(catalogJSON); err != nil {
		return html + gclog.Printf(gclog.LErrorLog,
			"Failed writing /%s/catalog.json: %s", board.Dir, err.Error()) + "<br />"
	}
	html += "/" + board.Dir + "/ built successfully."
	return
}

// BuildBoards builds the specified board IDs, or all boards if no arguments are passed
// The return value is a string of HTML with debug information produced by the build process.
func BuildBoards(which ...int) (html string) {
	var boards []gcsql.Board
	var err error
	if which == nil {
		boards = gcsql.AllBoards
	} else {
		for b, id := range which {
			boards = append(boards, gcsql.Board{})
			if err = boards[b].PopulateData(id); err != nil {
				return gclog.Printf(gclog.LErrorLog, "Error getting board information (ID: %d)", id)
			}
		}
	}
	if len(boards) == 0 {
		return "No boards to build."
	}

	for _, board := range boards {
		if err = buildBoard(&board, false, true); err != nil {
			return gclog.Printf(gclog.LErrorLog,
				"Error building /%s/: %s", board.Dir, err.Error()) + "<br />"
		}
		html += "Built /" + board.Dir + "/ successfully."
	}
	return
}

//BuildCatalog builds the catalog for a board with a given id
func BuildCatalog(boardID int) string {
	err := gctemplates.InitTemplates("catalog")
	if err != nil {
		return err.Error()
	}

	var board gcsql.Board
	if err = board.PopulateData(boardID); err != nil {
		return gclog.Printf(gclog.LErrorLog, "Error getting board information (ID: %d)", boardID)
	}

	catalogPath := path.Join(config.Config.DocumentRoot, board.Dir, "catalog.html")
	catalogFile, err := os.OpenFile(catalogPath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0777)
	if err != nil {
		return gclog.Printf(gclog.LErrorLog,
			"Failed opening /%s/catalog.html: %s", board.Dir, err.Error()) + "<br />"
	}

	threadOPs, err := gcsql.GetTopPosts(boardID, true)
	// threadOPs, err := getPostArr(map[string]interface{}{
	// 	"boardid":           boardID,
	// 	"parentid":          0,
	// 	"deleted_timestamp": nilTimestamp,
	// }, "ORDER BY bumped ASC")
	if err != nil {
		return gclog.Printf(gclog.LErrorLog,
			"Error building catalog for /%s/: %s", board.Dir, err.Error()) + "<br />"
	}

	var threadInterfaces []interface{}
	for _, thread := range threadOPs {
		threadInterfaces = append(threadInterfaces, thread)
	}

	if err = gcutil.MinifyTemplate(gctemplates.Catalog, map[string]interface{}{
		"boards":   gcsql.AllBoards,
		"config":   config.Config,
		"board":    board,
		"sections": gcsql.AllSections,
	}, catalogFile, "text/html"); err != nil {
		return gclog.Printf(gclog.LErrorLog,
			"Error building catalog for /%s/: %s", board.Dir, err.Error()) + "<br />"
	}
	return fmt.Sprintf("Built catalog for /%s/ successfully", board.Dir)
}

// Build builds the board and its thread files
// if newBoard is true, it adds a row to DBPREFIXboards and fails if it exists
// if force is true, it doesn't fail if the directories exist but does fail if it is a file
func buildBoard(board *gcsql.Board, newBoard bool, force bool) error {
	var err error
	if board.Dir == "" {
		return errors.New("board must have a directory before it is built")
	}
	if board.Title == "" {
		return errors.New("board must have a title before it is built")
	}

	dirPath := board.AbsolutePath()
	resPath := board.AbsolutePath("res")
	srcPath := board.AbsolutePath("src")
	thumbPath := board.AbsolutePath("thumb")
	dirInfo, _ := os.Stat(dirPath)
	resInfo, _ := os.Stat(resPath)
	srcInfo, _ := os.Stat(srcPath)
	thumbInfo, _ := os.Stat(thumbPath)
	if dirInfo != nil {
		if !force {
			return fmt.Errorf(pathExistsStr, dirPath)
		}
		if !dirInfo.IsDir() {
			return fmt.Errorf(dirIsAFileStr, dirPath)
		}
	} else {
		if err = os.Mkdir(dirPath, 0666); err != nil {
			return fmt.Errorf(genericErrStr, dirPath, err.Error())
		}
	}

	if resInfo != nil {
		if !force {
			return fmt.Errorf(pathExistsStr, resPath)
		}
		if !resInfo.IsDir() {
			return fmt.Errorf(dirIsAFileStr, resPath)
		}
	} else {
		if err = os.Mkdir(resPath, 0666); err != nil {
			return fmt.Errorf(genericErrStr, resPath, err.Error())
		}
	}

	if srcInfo != nil {
		if !force {
			return fmt.Errorf(pathExistsStr, srcPath)
		}
		if !srcInfo.IsDir() {
			return fmt.Errorf(dirIsAFileStr, srcPath)
		}
	} else {
		if err = os.Mkdir(srcPath, 0666); err != nil {
			return fmt.Errorf(genericErrStr, srcPath, err.Error())
		}
	}

	if thumbInfo != nil {
		if !force {
			return fmt.Errorf(pathExistsStr, thumbPath)
		}
		if !thumbInfo.IsDir() {
			return fmt.Errorf(dirIsAFileStr, thumbPath)
		}
	} else {
		if err = os.Mkdir(thumbPath, 0666); err != nil {
			return fmt.Errorf(genericErrStr, thumbPath, err.Error())
		}
	}

	if newBoard {
		board.CreatedOn = time.Now()
		err := gcsql.CreateBoard(board)
		if err != nil {
			return err
		}
	} else {
		if err = board.UpdateID(); err != nil {
			return err
		}
	}
	BuildBoardPages(board)
	BuildThreads(true, board.ID, 0)
	gcsql.ResetBoardSectionArrays()
	BuildFrontPage()
	if board.EnableCatalog {
		BuildCatalog(board.ID)
	}
	BuildBoardListJSON()
	return nil
}
