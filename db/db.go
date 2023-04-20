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
	"strings"
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
	err := DbPool.AutoMigrate(&models.User{}, &models.Contest{}, &models.Group{}, &models.Admin{}, &models.ModeratorContestId{}, &models.GroupContestId{}, &models.Moderators{}, &models.ModeratorGroup{})
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

func GetContestsInGroup(group int) ([]string, error) {
	var contests []models.GroupContestId
	raw := DbPool.Find(&contests, "group_contest LIKE ? AND belongs=1", strconv.Itoa(group)+"%")
	if errors.Is(raw.Error, gorm.ErrRecordNotFound) {
		return []string{}, nil
	} else if raw.Error != nil {
		return []string{}, raw.Error
	}
	var final []string
	for _, val := range contests {
		final = append(final, strings.Split(val.GroupContest, ",")[1])
	}
	return final, nil
}

func GetContestInfo(contest int) (models.BasicContest, error) {
	var result models.Contest
	res := DbPool.First(&result, contest)
	if errors.Is(res.Error, gorm.ErrRecordNotFound) {
		return models.BasicContest{}, nil
	} else if res.Error != nil {
		return models.BasicContest{}, res.Error
	}
	return models.BasicContest{Url: result.Url, ContestPicture: result.ContestPicture, Comment: result.Comment, StatementsUrl: result.StatementsUrl}, nil
}

func IsHostInGroup(group int, login string) bool {
	var id models.Moderators
	err := DbPool.First(&id, "login = ?", login)
	if err != nil {
		return false
	}
	var res models.ModeratorGroup
	err = DbPool.First(&res, "moderator_group_id LIKE ? AND is_host=1", strconv.Itoa(id.ID)+","+strconv.Itoa(group))
	if err != nil {
		return false
	}
	return true
}
