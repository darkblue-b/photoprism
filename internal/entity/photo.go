package entity

import (
	"sort"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/photoprism/photoprism/internal/form"
	"github.com/photoprism/photoprism/pkg/rnd"
	"github.com/photoprism/photoprism/pkg/txt"
	"github.com/ulule/deepcopier"
)

// Photo represents a photo that can have multiple image or sidecar files.
type Photo struct {
	ID                uint      `gorm:"primary_key"`
	TakenAt           time.Time `gorm:"type:datetime;index:idx_photos_taken_uuid;" json:"TakenAt"`
	PhotoUUID         string    `gorm:"type:varbinary(36);unique_index;index:idx_photos_taken_uuid;"`
	PhotoPath         string    `gorm:"type:varbinary(512);index;"`
	PhotoName         string    `gorm:"type:varbinary(256);"`
	PhotoTitle        string    `json:"PhotoTitle"`
	PhotoSubject      string    `json:"PhotoSubject"`
	PhotoKeywords     string    `json:"PhotoKeywords"`
	PhotoDescription  string    `gorm:"type:text;" json:"PhotoDescription"`
	PhotoNotes        string    `gorm:"type:text;" json:"PhotoNotes"`
	PhotoArtist       string    `json:"PhotoArtist"`
	PhotoCopyright    string    `json:"PhotoCopyright"`
	PhotoLicense      string    `json:"PhotoLicense"`
	PhotoFavorite     bool      `json:"PhotoFavorite"`
	PhotoPrivate      bool      `json:"PhotoPrivate"`
	PhotoNSFW         bool      `json:"PhotoNSFW"`
	PhotoStory        bool      `json:"PhotoStory"`
	PhotoLat          float64   `gorm:"index;" json:"PhotoLat"`
	PhotoLng          float64   `gorm:"index;" json:"PhotoLng"`
	PhotoAltitude     int       `json:"PhotoAltitude"`
	PhotoFocalLength  int       `json:"PhotoFocalLength"`
	PhotoIso          int       `json:"PhotoIso"`
	PhotoFNumber      float64   `json:"PhotoFNumber"`
	PhotoExposure     string    `gorm:"type:varbinary(64);" json:"PhotoExposure"`
	CameraID          uint      `gorm:"index:idx_photos_camera_lens;" json:"CameraID"`
	CameraSerial      string    `gorm:"type:varbinary(128);" json:"CameraSerial"`
	LensID            uint      `gorm:"index:idx_photos_camera_lens;" json:"LensID"`
	AccountID         uint      `json:"AccountID"`
	PlaceID           string    `gorm:"type:varbinary(16);index;" json:"PlaceID"`
	LocationID        string    `gorm:"type:varbinary(16);index;" json:"LocationID"`
	LocationEstimated bool      `json:"LocationEstimated"`
	PhotoCountry      string    `gorm:"index:idx_photos_country_year_month;" json:"PhotoCountry"`
	PhotoYear         int       `gorm:"index:idx_photos_country_year_month;"`
	PhotoMonth        int       `gorm:"index:idx_photos_country_year_month;"`
	TimeZone          string    `gorm:"type:varbinary(64);" json:"TimeZone"`
	TakenAtLocal      time.Time `gorm:"type:datetime;"`
	ModifiedTitle     bool      `json:"ModifiedTitle"`
	ModifiedDetails   bool      `json:"ModifiedDetails"`
	ModifiedLocation  bool      `json:"ModifiedLocation"`
	ModifiedDate      bool      `json:"ModifiedDate"`
	Camera            *Camera   `json:"Camera"`
	Lens              *Lens     `json:"Lens"`
	Location          *Location `json:"-"`
	Place             *Place    `json:"-"`
	Account           *Account  `json:"-"`
	Files             []File
	Labels            []PhotoLabel
	Keywords          []Keyword `json:"-"`
	Albums            []Album   `json:"-"`
	CreatedAt         time.Time
	UpdatedAt         time.Time
	DeletedAt         *time.Time `sql:"index"`
}

// SavePhoto updates a model using form data and persists it in the database.
func SavePhoto(model Photo, form form.Photo, db *gorm.DB) error {
	if err := deepcopier.Copy(&model).From(form); err != nil {
		return err
	}

	return db.Save(&model).Error
}

func (m *Photo) BeforeCreate(scope *gorm.Scope) error {
	if err := scope.SetColumn("PhotoUUID", rnd.PPID('p')); err != nil {
		return err
	}

	if m.TakenAt.IsZero() || m.TakenAtLocal.IsZero() {
		now := time.Now()

		if err := scope.SetColumn("TakenAt", now); err != nil {
			return err
		}

		if err := scope.SetColumn("TakenAtLocal", now); err != nil {
			return err
		}
	}

	return nil
}

func (m *Photo) BeforeSave(scope *gorm.Scope) error {
	if m.TakenAt.IsZero() || m.TakenAtLocal.IsZero() {
		now := time.Now()

		if err := scope.SetColumn("TakenAt", now); err != nil {
			return err
		}

		if err := scope.SetColumn("TakenAtLocal", now); err != nil {
			return err
		}
	}

	return nil
}

func (m *Photo) IndexKeywords(keywords []string, db *gorm.DB) {
	var keywordIds []uint

	// Add title, description and other keywords
	keywords = append(keywords, txt.Keywords(m.PhotoTitle)...)
	keywords = append(keywords, txt.Keywords(m.PhotoKeywords)...)
	keywords = append(keywords, txt.Keywords(m.PhotoSubject)...)
	keywords = append(keywords, txt.Keywords(m.PhotoArtist)...)
	keywords = append(keywords, txt.Keywords(m.PhotoDescription)...)
	last := ""

	sort.Strings(keywords)

	for _, w := range keywords {
		if len(w) < 3 || w == last {
			continue
		}

		last = w
		kw := NewKeyword(w).FirstOrCreate(db)

		if kw.Skip {
			continue
		}

		keywordIds = append(keywordIds, kw.ID)

		NewPhotoKeyword(m.ID, kw.ID).FirstOrCreate(db)
	}

	db.Where("photo_id = ? AND keyword_id NOT IN (?)", m.ID, keywordIds).Delete(&PhotoKeyword{})
}

func (m *Photo) PreloadFiles(db *gorm.DB) {
	q := db.NewScope(nil).DB().
		Table("files").
		Select(`files.*`).
		Where("files.photo_id = ?", m.ID).
		Order("files.file_primary DESC")

	logError(q.Scan(&m.Files))
}

/* func (m *Photo) PreloadLabels(db *gorm.DB) {
	q := db.NewScope(nil).DB().
		Table("labels").
		Select(`labels.*`).
		Joins("JOIN photos_labels ON photos_labels.label_id = labels.id AND photos_labels.photo_id = ?", m.ID).
		Where("labels.deleted_at IS NULL").
		Order("labels.label_name ASC")

	logError(q.Scan(&m.Labels))
} */

func (m *Photo) PreloadKeywords(db *gorm.DB) {
	q := db.NewScope(nil).DB().
		Table("keywords").
		Select(`keywords.*`).
		Joins("JOIN photos_keywords ON photos_keywords.keyword_id = keywords.id AND photos_keywords.photo_id = ?", m.ID).
		Order("keywords.keyword ASC")

	logError(q.Scan(&m.Keywords))
}

func (m *Photo) PreloadAlbums(db *gorm.DB) {
	q := db.NewScope(nil).DB().
		Table("albums").
		Select(`albums.*`).
		Joins("JOIN photos_albums ON photos_albums.album_uuid = albums.album_uuid AND photos_albums.photo_uuid = ?", m.PhotoUUID).
		Where("albums.deleted_at IS NULL").
		Order("albums.album_name ASC")

	logError(q.Scan(&m.Albums))
}

func (m *Photo) PreloadMany(db *gorm.DB) {
	m.PreloadFiles(db)
	// m.PreloadLabels(db)
	m.PreloadKeywords(db)
	m.PreloadAlbums(db)
}

func (m *Photo) NoLocation() bool {
	return m.LocationID == ""
}

func (m *Photo) HasLocation() bool {
	return m.LocationID != ""
}

func (m *Photo) NoPlace() bool {
	return len(m.PlaceID) < 2
}

func (m *Photo) HasPlace() bool {
	return len(m.PlaceID) >= 2
}

func (m *Photo) NoTitle() bool {
	return m.PhotoTitle == ""
}

func (m *Photo) NoDescription() bool {
	return m.PhotoDescription == ""
}

func (m *Photo) NoNotes() bool {
	return m.PhotoNotes == ""
}

func (m *Photo) NoArtist() bool {
	return m.PhotoArtist == ""
}

func (m *Photo) NoCopyright() bool {
	return m.PhotoCopyright == ""
}

func (m *Photo) NoSubject() bool {
	return m.PhotoSubject == ""
}

func (m *Photo) NoKeywords() bool {
	return m.PhotoKeywords == ""
}

func (m *Photo) NoCameraSerial() bool {
	return m.CameraSerial == ""
}

func (m *Photo) HasTitle() bool {
	return m.PhotoTitle != ""
}
