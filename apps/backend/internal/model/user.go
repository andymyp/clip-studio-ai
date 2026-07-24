package model

type User struct {
	Base
	Email string `gorm:"size:320;not null;uniqueIndex" json:"email"`
	Name  string `gorm:"size:160;not null" json:"name"`

	Videos     []Video     `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
	Watermarks []Watermark `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
}
