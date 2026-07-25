package model

type User struct {
	Base
	Email        string `gorm:"size:320;not null;uniqueIndex" json:"email"`
	PasswordHash string `gorm:"column:password_hash;size:60;not null" json:"-"`
	// Name remains internal for compatibility with databases created before
	// authentication was introduced.
	Name string `gorm:"size:160;not null;default:''" json:"-"`

	Videos []Video `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
}
