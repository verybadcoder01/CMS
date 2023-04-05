package models

type Config struct {
	EjudgeConAddress string `yaml:"ejudge_con_address"`
	DbPath           string `yaml:"db_path"`
	LogPath          string `yaml:"log_path"`
	Port             string `yaml:"port"`
}

type User struct {
	EjId           int    `gorm:"primaryKey;autoIncrement:false" json:"ejId"`
	FirstName      string `json:"firstName"`
	LastName       string `json:"lastName"`
	ProfilePicture string `json:"profilePicture"`
	Status         string `json:"status"`
}

// Group по факту это отдельная система. например, группа лкш2023, группа контестов 10и
type Group struct {
	ID   int `gorm:"primaryKey"`
	Name string
}

type BasicContest struct {
	Url            string `json:"url"`
	ContestPicture string `json:"contestPicture"`
	Comment        string `json:"comment"`
	StatementsUrl  string `json:"statementsUrl"`
}

type Contest struct {
	ID int `gorm:"primaryKey"` // делаем сами
	BasicContest
}

type Admin struct {
	ID          int `gorm:"primaryKey"`
	Description string
}

type GroupUserId struct {
	UserGroup string `gorm:"primaryKey;autoIncrement:false"`
	Role      Role   `gorm:"foreignKey:AdminRefer"`
}

type GroupContestId struct {
	GroupContest string `gorm:"primaryKey;autoIncrement:false"`
	Belongs      bool
}

type UserAndGroup struct {
	UserId  int  `json:"userId"`
	GroupId int  `json:"groupId"`
	Role    Role `json:"role"`
}

type Role int

const (
	NoAdmin Role = iota + 1
	YesAdmin
)

func (r Role) String() string {
	return []string{"не администратор", "администратор"}[r-1]
}
