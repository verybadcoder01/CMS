package db

import (
	"cms/models"
	"errors"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"log"
	"os"
	"strconv"
)

var DbPool *gorm.DB

func CreateDbFile(path string, p logger.Interface) {
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		_, err = os.Create(path)
		if err != nil {
			log.Fatal(err)
		}
		DbPool, err = gorm.Open(sqlite.Open(path), &gorm.Config{Logger: p})
		log.Println("Database created successfully!")
		InitTables()
	} else {
		DbPool, err = gorm.Open(sqlite.Open(path), &gorm.Config{Logger: p})
		if err != nil {
			log.Fatal(err)
		}
	}
}

func InitTables() {
	err := DbPool.AutoMigrate(&models.User{}, &models.Contest{}, &models.Group{}, &models.Admin{}, &models.UserContestId{}, &models.GroupContestId{}, &models.Moderators{}, &models.ModeratorGroup{})
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Tables created successfully!")
	// (1, NoAdmin), (2, YesAdmin)
	DbPool.Select("description").Create(&models.Admin{Description: models.NoAdmin.String()})
	DbPool.Select("description").Create(&models.Admin{Description: models.YesAdmin.String()})
}

func AddContestToGroup(GroupId int, contestId int) {
	var idMixed = strconv.Itoa(GroupId) + "," + strconv.Itoa(contestId)
	var existing models.GroupContestId
	res := DbPool.First(&existing, "group_contest = ?", idMixed)
	if errors.Is(res.Error, gorm.ErrRecordNotFound) {
		DbPool.Create(&models.GroupContestId{GroupContest: idMixed, Belongs: true})
	} else {
		existing.Belongs = true
		DbPool.Save(&existing)
	}
}

func AddUserToContest(ContestId int, userId int, role models.Role) error {
	var idMixed = strconv.Itoa(userId) + "," + strconv.Itoa(ContestId)
	var existing models.UserContestId
	res := DbPool.First(&existing, "user_contest = ?", idMixed)
	if errors.Is(res.Error, gorm.ErrRecordNotFound) {
		res = DbPool.Create(&models.UserContestId{UserContest: idMixed, Role: role})
	} else {
		existing.Role = role
		res = DbPool.Save(&existing)
	}
	return res.Error
}

func AddContest(contest models.BasicContest) error {
	res := DbPool.Create(&models.Contest{BasicContest: models.BasicContest{Url: contest.Url, ContestPicture: contest.ContestPicture, StatementsUrl: contest.StatementsUrl, Comment: contest.Comment}})
	return res.Error
}

func CreateUser(user models.User) error {
	res := DbPool.Create(&user)
	return res.Error
}

func CreateModerator(moderator models.SimpleModerator) error {
	var dest models.Moderators
	res := DbPool.First(&dest, "login = ?", moderator.Login)
	if errors.Is(res.Error, gorm.ErrRecordNotFound) {
		res = DbPool.Create(&models.Moderators{SimpleModerator: models.SimpleModerator{Login: moderator.Login, Password: moderator.Password}})
	}
	return res.Error
}

func GetPasswordHash(login string) (string, error) {
	var password models.Moderators
	res := DbPool.First(&password, "login = ?", login)
	if errors.Is(res.Error, gorm.ErrRecordNotFound) {
		return "", res.Error
	} else {
		return password.Password, nil
	}
}
