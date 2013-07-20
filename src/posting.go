// functions for handling posting, uploading, and post/thread/board page building

package main

import (
	"net/http"
	"io/ioutil"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"image/gif"
	"image/png"
	"math/rand"
	"os"
	"path"
	"regexp"
	"./lib/resize"
	"database/sql"
	"strconv"
	"strings"
	"syscall"
	"time"
)

var (
	UnsupportedFiletypeError =  errors.New("Upload filetype not supported")
	FileWriteError = errors.New("Couldn't write file")
	TemplateExecutionError = errors.New("Failed executing template")
	last_post PostTable
)

func generateTripCode(input string) string {
	re := regexp.MustCompile("/[^.-z]/")
	input = string(re.ReplaceAllLiteral([]byte(input), []byte(".")))
	input += "   " //padding
	salt := byteByByteReplace(input[1:3],":;<=>?@[\\]^_`", "ABCDEFGabcdef") // stole-I MEAN BORROWED from Kusaba X
	return crypt(input,salt)[3:]
}

func buildBoardPages(boardid int, boards []BoardsTable, sections []interface{}) (html string) {
	var board BoardsTable
	for b,_ := range boards {
		if boards[b].ID == boardid {
			board = boards[b]
		}
	}

	var interfaces []interface{}
	var threads []interface{}
	op_posts,err := getPostArr("SELECT * FROM `"+config.DBprefix+"posts` WHERE `boardid` = "+strconv.Itoa(board.ID)+" AND `parentid` = 0 ORDER BY `bumped` ASC LIMIT "+strconv.Itoa(config.ThreadsPerPage_img))
	if err != nil {
		html += err.Error() + "<br />"
		op_posts = make([]interface{},0)
	}

	// yes I know there's a better way to do this, minimizing the number of sql statements made, sorting and splitting it in the code, but I'll fix that later
	for _,op_post_i := range op_posts {
		var thread Thread
		var posts_in_thread []interface{}
		op_post := op_post_i.(PostTable)
		thread.IName = "thread"

		if op_post.Stickied {
			posts_in_thread,err = getPostArr("SELECT * FROM `"+config.DBprefix+"posts` WHERE `boardid` = "+strconv.Itoa(board.ID)+" AND (`id` = "+strconv.Itoa(op_post.ID)+" OR `parentid` = "+strconv.Itoa(op_post.ID)+") LIMIT "+strconv.Itoa(config.StickyRepliesOnBoardPage))
		} else {
			posts_in_thread,err = getPostArr("SELECT * FROM `"+config.DBprefix+"posts` WHERE `boardid` = "+strconv.Itoa(board.ID)+" AND (`id` = "+strconv.Itoa(op_post.ID)+" OR `parentid` = "+strconv.Itoa(op_post.ID)+") LIMIT "+strconv.Itoa(config.RepliesOnBoardpage))							
		}

		if err != nil {
			html += err.Error()+"<br />"
			posts_in_thread = make([]interface{},0)
		} else {
			thread.Posts = posts_in_thread
		}
		threads = append(threads, thread)
	}

    interfaces = append(interfaces, config)

    var boards_i []interface{}
    for _,b := range boards {
    	boards_i = append(boards_i,b)
    }
    var boardinfo_i []interface{}
    boardinfo_i = append(boardinfo_i,board)

    interfaces = append(interfaces, &Wrapper{IName: "boards", Data: boards_i})
    interfaces = append(interfaces, &Wrapper{IName: "sections", Data: sections})
    interfaces = append(interfaces, &Wrapper{IName: "threads", Data: threads})
    interfaces = append(interfaces, &Wrapper{IName: "boardinfo", Data: boardinfo_i})

	wrapped := &Wrapper{IName: "boardpage",Data: interfaces}
	os.Remove(path.Join(config.DocumentRoot,board.Dir,"board.html"))
	
	board_file,err := os.OpenFile(path.Join(config.DocumentRoot,board.Dir,"board.html"),os.O_CREATE|os.O_RDWR,0777)
	if err != nil {
		html += err.Error()
	}

	defer func() {
		if uhoh, ok := recover().(error); ok {
			error_log.Write(TemplateExecutionError.Error())
			fmt.Println(uhoh.Error())
		}
	}()
	err = img_boardpage_tmpl.Execute(board_file,wrapped)
	if err != nil {
		html += "Failed building /"+board.Dir+"/: "+err.Error()
		error_log.Write(err.Error())
	} else {
		html += "/"+board.Dir+"/ built successfully.<br />"
	}
	return
}

func buildThread(op_post PostTable, is_reply bool) (err error) {
	var op_id string
	if is_reply {
		op_id = strconv.Itoa(op_post.ParentID)
	} else {
		op_id = strconv.Itoa(op_post.ID)
	}
	thread_posts,err := getPostArr("SELECT * FROM `ponychan_bunker_posts` WHERE `deleted_timestamp` = '"+nil_timestamp+"' AND (`parentid` = "+op_id+" OR `id` = "+op_id+") AND `boardid` = "+strconv.Itoa(op_post.BoardID))
	if err != nil {
		exitWithErrorPage(writer,err.Error())
	}
	board_arr := getBoardArr("")
	sections_arr := getSectionArr("")

	var board_dir string
	for _,board_i := range board_arr {
		board := board_i
		if board.ID == op_post.BoardID {
			board_dir = board.Dir

			break
		}
	}

    var interfaces []interface{}
    interfaces = append(interfaces, config)
    interfaces = append(interfaces, thread_posts)
    var board_arr_i []interface{}
    for _,b := range board_arr {
    	board_arr_i = append(board_arr_i,b)
    }
    interfaces = append(interfaces, &Wrapper{IName:"boards", Data: board_arr_i})
    interfaces = append(interfaces, &Wrapper{IName:"sections", Data: sections_arr})

	wrapped := &Wrapper{IName: "threadpage",Data: interfaces}
	os.Remove(path.Join(config.DocumentRoot,board_dir+"/res/"+op_id+".html"))
	
	thread_file,err := os.OpenFile(path.Join(config.DocumentRoot,board_dir+"/res/"+op_id+".html"),os.O_CREATE|os.O_RDWR,0777)

	if err != nil {
		return err
	}

	defer func() {
		if _, ok := recover().(error); ok {
			error_log.Write(TemplateExecutionError.Error())
		}
	}()
	err = img_thread_tmpl.Execute(thread_file,wrapped)
	return err
}

// checks to see if the poster's tripcode/name is banned, if the IP is banned, or if the file checksum is banned
func checkBannedStatus(post PostTable) bool {
	return false
}

type ThumbnailPre struct {
	Filename_old string
	Filename_new string
	Filepath string
	Width int
	Height int
	Obj image.Image
	ThumbObj image.Image
}

func loadImage(file *os.File) (image.Image,error) {
	filetype := file.Name()[len(file.Name())-3:]
	var image_obj image.Image
	var err error

	if filetype == "gif" {
		image_obj,err = gif.Decode(file)
	} else if filetype == "jpeg" || filetype == "jpg" {
		image_obj,err = jpeg.Decode(file)
	} else if filetype == "png" {
		image_obj,err = png.Decode(file)
	} else {
		image_obj = nil
		err = UnsupportedFiletypeError
	}
	return image_obj,err
}

func createThumbnail(image_obj image.Image, size string) image.Image {
	var thumb_width int
	var thumb_height int

	switch {
		case size == "op":
			thumb_width = config.ThumbWidth
			thumb_height = config.ThumbHeight
		case size == "reply":
			thumb_width = config.ThumbWidth_reply
			thumb_height = config.ThumbHeight_reply
		case size == "catalog":
			thumb_width = config.ThumbWidth_catalog
			thumb_height = config.ThumbHeight_catalog
	}
	old_rect := image_obj.Bounds()
	if thumb_width >= old_rect.Max.X && thumb_height >= old_rect.Max.Y {
		return image_obj
	}
	
	thumb_w,thumb_h := getThumbnailSize(old_rect.Max.X,old_rect.Max.Y,size)
	image_obj = resize.Resize(image_obj, image.Rect(0,0,old_rect.Max.X,old_rect.Max.Y), thumb_w,thumb_h)
	return image_obj
}


func getFiletype(name string) string {
	filetype := strings.ToLower(name[len(name)-4:])
	if filetype == ".gif" {
		return "gif"
	} else if filetype == ".jpg" || filetype == "jpeg" {
		return "jpg"
	} else if filetype == ".png" {
		return "png"
	} else {
		return name[len(name)-3:]
	}
}

func getNewFilename() string {
	now := time.Now().Unix()
	rand.Seed(now)
	return strconv.Itoa(int(now))+strconv.Itoa(int(rand.Intn(98)+1))
}

// find out what out thumbnail's width and height should be, partially ripped from Kusaba X
func getThumbnailSize(w int, h int,size string) (new_w int, new_h int) {
	var thumb_width int
	var thumb_height int

	switch {
		case size == "op":
			thumb_width = config.ThumbWidth
			thumb_height = config.ThumbHeight
		case size == "reply":
			thumb_width = config.ThumbWidth_reply
			thumb_height = config.ThumbHeight_reply
		case size == "catalog":
			thumb_width = config.ThumbWidth_catalog
			thumb_height = config.ThumbHeight_catalog
	}
	if w == h {
		new_w = thumb_width
		new_h = thumb_height
	} else {
		var percent float32
		if (w > h) {
			percent = float32(thumb_width) / float32(w)
		} else {
			percent = float32(thumb_height) / float32(h)
		}
		new_w = int(float32(w) * percent)
		new_h = int(float32(h) * percent)
	}
	return
}

// inserts prepared post object into the SQL table so that it can be rendered
func insertPost(writer *http.ResponseWriter, post PostTable,bump bool) error {

	post_sql_str := "INSERT INTO `"+config.DBprefix+"posts` (`boardid`,`parentid`,`name`,`tripcode`,`email`,`subject`,`message`,`password`"
	if post.Filename != "" {
		post_sql_str += ",`filename`,`filename_original`,`file_checksum`,`filesize`,`image_w`,`image_h`,`thumb_w`,`thumb_h`"
	}
	post_sql_str += ",`ip`"
	post_sql_str += ",`timestamp`,`poster_authority`,`stickied`,`locked`) VALUES("+strconv.Itoa(post.BoardID)+","+strconv.Itoa(post.ParentID)+",'"+post.Name+"','"+post.Tripcode+"','"+post.Email+"','"+post.Subject+"','"+post.Message+"','"+post.Password+"'"
	if post.Filename != "" {
		post_sql_str += ",'"+post.Filename+"','"+post.FilenameOriginal+"','"+post.FileChecksum+"',"+strconv.Itoa(int(post.Filesize))+","+strconv.Itoa(post.ImageW)+","+strconv.Itoa(post.ImageH)+","+strconv.Itoa(post.ThumbW)+","+strconv.Itoa(post.ThumbH)
	}
	post_sql_str += ",'"+post.IP+"','"+getSpecificSQLDateTime(post.Timestamp)+"',"+strconv.Itoa(post.PosterAuthority)+","
	if post.Stickied {
		post_sql_str += "1,"
	} else {
		post_sql_str += "0,"
	}
	if post.Locked {
		post_sql_str += "1);"
	} else {
		post_sql_str += "0);"
	}
	_,err := db.Exec(post_sql_str)

	if err != nil {
		exitWithErrorPage(*writer,err.Error())
	}
	return nil
}


func makePost(w http.ResponseWriter, r *http.Request) {
	request = *r
	writer = w
	
	var post PostTable
	post.IName = "post"
	post.ParentID,_ = strconv.Atoi(request.FormValue("threadid"))
	post.BoardID,_ = strconv.Atoi(request.FormValue("boardid"))

	var count int
	var postid int
	var boardid int

	err := db.QueryRow("SELECT (SELECT COUNT(*) FROM `"+config.DBprefix+"posts` WHERE `boardid` = "+strconv.Itoa(post.BoardID)+") AS `count`, `"+config.DBprefix+"posts`.`id` AS `id`, `"+config.DBprefix+"boards`.`id` AS `boardid` FROM `"+config.DBprefix+"posts`, `"+config.DBprefix+"boards` WHERE `boardid` = "+strconv.Itoa(post.BoardID)+" ORDER BY `"+config.DBprefix+"posts`.`id` DESC LIMIT 1").Scan(&count,&postid,&boardid)
	
	if err != nil {
		if err == sql.ErrNoRows {
			count = 0
		} else {
			error_log.Write(err.Error())
			exitWithErrorPage(w, err.Error())
			return
		}
	}

	if count == 0 {
		var first_post int
		err = db.QueryRow("SELECT `first_post` FROM `"+config.DBprefix+"boards` WHERE `id` = "+strconv.Itoa(post.BoardID)+" LIMIT 1").Scan(&first_post)
		if err != nil {
			error_log.Write(err.Error())
			exitWithErrorPage(w, err.Error())
		}
		post.ID = first_post
	} else {
		post.ID = postid + 1
	}
	
	post_name := escapeString(request.FormValue("postname"))
	if strings.Index(post_name, "#") == -1 {
		post.Name = post_name
	} else if strings.Index(post_name, "#") == 0 {
		post.Tripcode = generateTripCode(post_name[1:])
	} else if strings.Index(post_name, "#") > 0 {
		post_name_arr := strings.SplitN(post_name,"#",2)
		post.Name = post_name_arr[0]
		post.Tripcode = generateTripCode(post_name_arr[1])
	}
	
	email_command := ""
	post_email := escapeString(request.FormValue("postemail"))
	if strings.Index(post_email, "#") == -1 {
		post.Email = post_email
	} else if strings.Index(post_email, "#") == 0 {
		email_command = post_email[1:]
	} else if strings.Index(post_email, "#") > 0 {
		post_email_arr := strings.SplitN(post_email,"#",2)
		post.Email = post_email_arr[0]
		email_command = post_email_arr[1]
	}

	post.Subject = escapeString(request.FormValue("postsubject"))
	post.Message = escapeString(request.FormValue("postmsg"))
	post.Password = md5_sum(request.FormValue("postpassword"))
	http.SetCookie(writer, &http.Cookie{Name: "name", Value: post.Name, Path: "/", Domain: config.Domain, RawExpires: getSpecificSQLDateTime(time.Now().Add(time.Duration(31536000))),MaxAge: 31536000})
	http.SetCookie(writer, &http.Cookie{Name: "email", Value: post.Email, Path: "/", Domain: config.Domain, RawExpires: getSpecificSQLDateTime(time.Now().Add(time.Duration(31536000))),MaxAge: 31536000})
	http.SetCookie(writer, &http.Cookie{Name: "password", Value: request.FormValue("postpassword"), Path: "/", Domain: config.Domain, RawExpires: getSpecificSQLDateTime(time.Now().Add(time.Duration(31536000))),MaxAge: 31536000})	
	post.IP = request.RemoteAddr
	post.Timestamp = time.Now()
	post.PosterAuthority = getStaffRank()
	post.Bumped = time.Now()
	post.Stickied = request.FormValue("modstickied") == "on"
	post.Locked = request.FormValue("modlocked") == "on"

	//post has no referrer, or has a referrer from a different domain, probably a spambot
	if !validReferrer(request) {
		access_log.Write("Rejected post from possible spambot @ : "+request.RemoteAddr)
		//TODO: insert post into temporary post table and add to report list
	}
	file,handler,uploaderr := request.FormFile("imagefile")
	if uploaderr != nil {
		// no file was uploaded
		fmt.Println(uploaderr.Error())
		post.Filename = ""
		access_log.Write("Receiving post from "+request.RemoteAddr+", referred from: "+request.Referer())

	} else {
		data,err := ioutil.ReadAll(file)
		if err != nil {
			exitWithErrorPage(w,"Couldn't read file")
		} else {
			post.FilenameOriginal = handler.Filename
			filetype := getFiletype(post.FilenameOriginal)
			thumb_filetype := filetype
			if thumb_filetype == "gif" {
				thumb_filetype = "jpg"
			}

			post.Filename = getNewFilename()+"."+getFiletype(post.FilenameOriginal)
			board_dir := getBoardArr("`id` = "+request.FormValue("boardid"))[0].Dir
			file_path := path.Join(config.DocumentRoot,"/"+board_dir+"/src/",post.Filename)
			thumb_path := path.Join(config.DocumentRoot,"/"+board_dir+"/thumb/",strings.Replace(post.Filename,"."+filetype,"t."+thumb_filetype,-1))

			err := ioutil.WriteFile(file_path, data, 0777)
			if err != nil {
				exitWithErrorPage(w,"Couldn't write file.")
			}

			image_file,err := os.OpenFile(file_path, os.O_RDONLY, 0)
			if err != nil {
				exitWithErrorPage(w,"Couldn't read saved file")
			}
			
			img,err := loadImage(image_file)
			if err != nil {
				exitWithErrorPage(w,err.Error())
			} else {
				//post.FileChecksum string
				stat,err := image_file.Stat()
				if err != nil {
					exitWithErrorPage(w,err.Error())
				} else {
					post.Filesize = int(stat.Size())
				}
				post.ImageW = img.Bounds().Max.X
				post.ImageH = img.Bounds().Max.Y
				if post.ParentID == 0 {
					post.ThumbW,post.ThumbH = getThumbnailSize(post.ImageW,post.ImageH,"op")	
				} else {
					post.ThumbW,post.ThumbH = getThumbnailSize(post.ImageW,post.ImageH,"reply")	
				}
				

				access_log.Write("Receiving post with image: "+handler.Filename+" from "+request.RemoteAddr+", referrer: "+request.Referer())

				if(request.FormValue("spoiler") == "on") {
					_,err := os.Stat(path.Join(config.DocumentRoot,"spoiler.png"))
					if err != nil {
						exitWithErrorPage(w,"missing /spoiler.png")
					} else {
						err = syscall.Symlink(path.Join(config.DocumentRoot,"spoiler.png"),thumb_path)
						if err != nil {
							exitWithErrorPage(w,err.Error())
						}
					}
				} else 	if config.ThumbWidth >= post.ImageW && config.ThumbHeight >= post.ImageH {
					post.ThumbW = img.Bounds().Max.X
					post.ThumbH = img.Bounds().Max.Y
					err := syscall.Symlink(file_path,thumb_path)
					if err != nil {
						exitWithErrorPage(w,err.Error())
					}
				} else {
					var thumbnail image.Image
					if post.ParentID == 0 {
						thumbnail = createThumbnail(img,"op")
					} else {
						thumbnail = createThumbnail(img,"reply")
					}
					err = saveImage(thumb_path, &thumbnail)
					if err != nil {
						exitWithErrorPage(w,err.Error())
					} else {
						
					}
				}
			}
		}
	}

	if post.Message == "" && post.Filename == "" {
		exitWithErrorPage(w,"Post must contain a message if no image is uploaded.")
	}
	insertPost(&w, post,email_command != "sage")
	if post.ParentID > 0 {
		post_arr,err := getPostArr("SELECT * FROM `ponychan_bunker_posts` WHERE `deleted_timestamp` = '"+nil_timestamp+"' AND `parentid` = "+strconv.Itoa(post.ParentID)+" AND `boardid` = "+strconv.Itoa(post.BoardID))
		if err != nil {
			exitWithErrorPage(writer,err.Error())
		}

		buildThread(post_arr[0].(PostTable),true)
	} else {
		buildThread(post,false)
	}
	boards := getBoardArr("")
	sections := getSectionArr("")
	buildBoardPages(post.BoardID, boards, sections)
	if email_command == "noko" {
		if post.ParentID == 0 {
			http.Redirect(writer,&request,"/test/res/"+strconv.Itoa(post.ID)+".html",http.StatusFound)
		} else {
			http.Redirect(writer,&request,"/test/res/"+strconv.Itoa(post.ParentID)+".html",http.StatusFound)
		}
	} else {
		http.Redirect(writer,&request,"/test/",http.StatusFound)
	}
}


func shortenPostForBoardPage(post *string) {

}


func saveImage(path string, image_obj *image.Image) error {
	outwriter,err := os.OpenFile(path, os.O_RDWR|os.O_CREATE,0777)
	if err == nil {
		filetype := path[len(path)-4:]
		if filetype == ".gif" {
			//because Go doesn't come with a GIF writer :c
			jpeg.Encode(outwriter, *image_obj, &jpeg.Options{Quality: 80})
		} else if filetype == ".jpg" || filetype == "jpeg" {
			jpeg.Encode(outwriter, *image_obj, &jpeg.Options{Quality: 80})
		} else if filetype == ".png" {
			png.Encode(outwriter, *image_obj)
		} else {
			return UnsupportedFiletypeError
		}
	}
	return err
}
