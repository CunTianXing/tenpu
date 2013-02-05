package tenpu

import (
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"path"
	"time"
)

type BlobStorage interface {
	Put(filename string, contentType string, body io.Reader, attachment *Attachment) (err error)
	Delete(attachment *Attachment) (err error)
	Copy(attachment *Attachment, w io.Writer) (err error)
	// Find(collectionName string, query interface{}, result interface{}) (err error)
	Zip(attachments []*Attachment, w io.Writer) (err error)
}

type MetaStorage interface {
	Put(att *Attachment) (err error)
	Remove(id string) (err error)
	Attachments(ownerid string) (r []*Attachment)
	AttachmentsByOwnerIds(ownerids []string) (r []*Attachment)
	AttachmentsCountByOwnerIds(ownerids []string) (r int)
	AttachmentById(id string) (r *Attachment)
	AttachmentByIds(ids []string) (r []*Attachment)
	AttachmentsByGroupId(groupId string) (r *Attachment)
}

type StorageMaker interface {
	Make(r *http.Request) (blob BlobStorage, meta MetaStorage, input Input, err error)
}

type Attachment struct {
	Id            string `bson:"_id"`
	OwnerId       []string
	Category      string
	Filename      string
	ContentType   string
	MD5           string
	ContentLength int64
	Error         string
	GroupId       []string
	UploadTime    time.Time
	Width         int
	Height        int
}

func (att *Attachment) MakeId() interface{} {
	return att.Id
}

func (att *Attachment) IsImage() (r bool) {
	switch att.ContentType {
	default:
		r = false
	case "image/png", "image/jpeg", "image/gif":
		r = true
	}
	return

}

func (att *Attachment) Extname() (r string) {
	r = path.Ext(att.Filename)
	if len(r) > 0 {
		r = r[1:]
	}
	return
}

type Input interface {
	Get() (id string, filename string, contentType string, thumb string, download bool)
	SetAttrsForDelete(att *Attachment) (shouldUpdate bool, shouldDelete bool, err error)
	SetAttrsForCreate(att *Attachment) (err error)
	SetMultipart(part *multipart.Part) (isFile bool)
	LoadAttachments() (r []*Attachment, err error)
}

func DeleteAttachment(input Input, blob BlobStorage, meta MetaStorage) (r []*Attachment, err error) {

	id, _, _, _, _ := input.Get()

	if id == "" {
		err = errors.New("id required.")
		return
	}

	att := meta.AttachmentById(id)

	shouldUpdate, _, err := input.SetAttrsForDelete(att)

	if err != nil {
		return
	}

	if shouldUpdate {
		err = meta.Put(att)
		r = []*Attachment{att}
		return
	}

	err = blob.Delete(att)
	if err != nil {
		return
	}

	err = meta.Remove(id)
	if err != nil {
		return
	}

	r = []*Attachment{att}
	return

}

func CreateAttachment(input Input, blob BlobStorage, meta MetaStorage, body io.Reader) (att *Attachment, err error) {
	att = &Attachment{}
	err = input.SetAttrsForCreate(att)

	if err != nil {
		return
	}

	_, filename, contentType, _, _ := input.Get()

	att.UploadTime = time.Now()

	err = blob.Put(filename, contentType, body, att)
	if err != nil {
		return
	}

	err = meta.Put(att)
	if err != nil {
		return
	}

	return
}
