package tenpu

import (
	"encoding/json"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"time"
)

type Result struct {
	Error       string
	Attachments []*Attachment
}

func MakeFileLoader(maker StorageMaker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		storage, meta, input, err := maker.Make(r)

		id, _, _, _, download := input.Get()
		if id == "" || err != nil {
			http.NotFound(w, r)
			return
		}

		att := meta.AttachmentById(id)
		if att == nil {
			http.NotFound(w, r)
			return
		}

		if download {
			w.Header().Set("Content-Type", "application/octet-stream")
		} else {
			w.Header().Set("Content-Type", att.ContentType)
		}
		w.Header().Set("Content-Length", fmt.Sprintf("%d", att.ContentLength))
		SetCacheControl(w, 30)
		err = storage.Copy(att, w)

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		return
	}
}

func MakeZipFileLoader(maker StorageMaker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		storage, _, input, err := maker.Make(r)
		var atts []*Attachment
		atts, err = input.LoadAttachments()

		if atts == nil {
			http.NotFound(w, r)
			return
		}
		// w.Header().Set("Content-Type", "application/zip")
		// w.Header().Set("Content-Length", fmt.Sprintf("%d", att.ContentLength))
		// w.Header().Set("Expires", formatDays(30))
		// w.Header().Set("Cache-Control", "max-age="+formatDayToSec(30))

		err = storage.Zip(atts, w)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		return
	}
}

func MakeDeleter(maker StorageMaker) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		blob, meta, input, err := maker.Make(r)

		atts, err := DeleteAttachment(input, blob, meta)

		if err != nil {
			writeJson(w, err.Error(), atts)
			return
		}

		writeJson(w, "", atts)
		return
	}
}

func MakeUploader(maker StorageMaker) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		blob, meta, input, err1 := maker.Make(r)
		if err1 != nil {
			writeJson(w, err1.Error(), nil)
			return
		}

		mr, err := r.MultipartReader()

		if err != nil {
			writeJson(w, err.Error(), nil)
			return
		}

		var part *multipart.Part
		var attachments []*Attachment

		for {
			part, err = mr.NextPart()
			if err != nil {
				break
			}

			isFile := input.SetMultipart(part)
			if !isFile {
				continue
			}

			var att *Attachment
			att, err = CreateAttachment(input, blob, meta, part)
			if err != nil {
				att.Error = err.Error()
			}
			attachments = append(attachments, att)
		}

		if len(attachments) == 0 {
			writeJson(w, "No attachments uploaded.", nil)
			return
		}

		for _, att := range attachments {
			if att.Error != "" {
				err = errors.New("Some attachment has error")
				break
			}
		}

		if err != nil {
			writeJson(w, err.Error(), attachments)
			return
		}

		ats := meta.AttachmentsByOwnerIds(attachments[0].OwnerId)
		writeJson(w, "", ats)
		return
	}
}

func SetCacheControl(w http.ResponseWriter, days int) {
	w.Header().Set("Expires", formatDays(days))
	w.Header().Set("Cache-Control", "max-age="+formatDayToSec(days))
}

func writeJson(w http.ResponseWriter, err string, attachments []*Attachment) {
	r := &Result{
		Error:       err,
		Attachments: attachments,
	}
	w.Header().Set("Content-Type", "application/json")
	b, _ := json.Marshal(r)
	w.Write(b)
}

// the time format used for HTTP headers 
const httpTimeFormat = "Mon, 02 Jan 2006 15:04:05 GMT"

func formatHour(hours string) string {
	d, _ := time.ParseDuration(hours + "h")
	return time.Now().Add(d).Format(httpTimeFormat)
}

func formatDays(day int) string {
	return formatHour(fmt.Sprintf("%d", day*24))
}

func formatDayToSec(day int) string {
	return fmt.Sprintf("%d", day*60*60*24)
}
