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

func CreateDbFile(path string, p logger.Interface, defaultAdmin models.SimpleModerator) {
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		_, err = os.Create(path)
		if err != nil {
			log.Fatal(err)
		}
		DbPool, err = gorm.Open(sqlite.Open(path), &gorm.Config{Logger: p})
		log.Println("Database created successfully!")
		InitTables(defaultAdmin)
	} else {
		DbPool, err = gorm.Open(sqlite.Open(path), &gorm.Config{Logger: p})
		if err != nil {
			log.Fatal(err)
		}
	}
}

func InitTables(defaultAdmin models.SimpleModerator) {
	err := DbPool.AutoMigrate(&models.User{}, &models.Contest{}, &models.Group{}, &models.Admin{}, &models.GroupContestId{}, &models.Moderators{}, &models.ModeratorGroup{})
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Tables created successfully!")
	// (1, NoAdmin), (2, YesAdmin)
	DbPool.Select("description").Create(&models.Admin{Description: models.NoAdmin.String()})
	DbPool.Select("description").Create(&models.Admin{Description: models.YesAdmin.String()})
	isOk := CreateModerator(models.SimpleModerator{Login: defaultAdmin.Login, Password: defaultAdmin.Password})
	if isOk == false {
		log.Fatal("kakoy-to pizdec")
	}
}

func AddContestToGroup(GroupId int, contestId int) error {
	var idMixed = strconv.Itoa(GroupId) + "," + strconv.Itoa(contestId)
	var existing models.GroupContestId
	res := DbPool.First(&existing, "group_contest = ?", idMixed)
	if errors.Is(res.Error, gorm.ErrRecordNotFound) {
		DbPool.Create(&models.GroupContestId{GroupContest: idMixed, Belongs: true})
	} else if res.Error == nil {
		existing.Belongs = true
		DbPool.Save(&existing)
	} else {
		return res.Error
	}
	return nil
}

func AddContest(contest models.BasicContest) error {
	newContest := models.Contest{BasicContest: contest}
	res := DbPool.First(&newContest, "name = ? AND url = ? AND contest_picture = ? AND comment = ? AND statements_url = ?", contest.Name, contest.Url, contest.ContestPicture, contest.Comment, contest.StatementsUrl)
	if errors.Is(res.Error, gorm.ErrRecordNotFound) {
		res = DbPool.Create(&models.Contest{BasicContest: contest})
		return res.Error
	} else if res.Error == nil && newContest.BasicContest == contest {
		return gorm.ErrInvalidData
	}
	return nil
}

func AddGroup(group models.BasicGroup) error {
	res := DbPool.Create(&models.Group{BasicGroup: group})
	return res.Error
}

// AddHostToGroup выдает админку или создает запись, если ее нет
func AddHostToGroup(GroupId int, ModeratorId int) error {
	var idMixed = strconv.Itoa(ModeratorId) + "," + strconv.Itoa(GroupId)
	var existing models.ModeratorGroup
	res := DbPool.First(&existing, "moderator_group_id = ?", idMixed)
	if errors.Is(res.Error, gorm.ErrRecordNotFound) {
		DbPool.Create(&models.ModeratorGroup{ModeratorGroupId: idMixed, IsHost: true})
	} else if res.Error == nil {
		existing.IsHost = true
		DbPool.Save(&existing)
	} else {
		return res.Error
	}
	return nil
}

func GetContestId(find models.BasicContest) (int, error) {
	var res models.Contest
	err := DbPool.Find(&res, "name = ? AND url = ? AND contest_picture = ? AND comment = ? AND statements_url = ?", find.Name, find.Url, find.ContestPicture, find.Comment, find.StatementsUrl)
	return res.ID, err.Error
}

func GetGroupId(name string) (int, error) {
	var res models.Group
	err := DbPool.Find(&res, "name = ?", name)
	return res.ID, err.Error
}

func GetModeratorId(login string) (int, error) {
	var res models.Moderators
	err := DbPool.First(&res, "login = ?", login)
	return res.ID, err.Error
}

func CreateUser(user models.User) error {
	res := DbPool.Create(&user)
	return res.Error
}

func CreateModerator(moderator models.SimpleModerator) bool {
	var dest models.Moderators
	res := DbPool.First(&dest, "login = ?", moderator.Login)
	if errors.Is(res.Error, gorm.ErrRecordNotFound) {
		res = DbPool.Create(&models.Moderators{SimpleModerator: models.SimpleModerator{Login: moderator.Login, Password: moderator.Password}})
	} else if res.Error != nil {
		return false
	}
	if dest.Login == moderator.Login {
		return false
	}
	return true
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
	raw := DbPool.Find(&contests, "group_contest LIKE ? AND belongs=1", strconv.Itoa(group)+",%")
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
	return result.BasicContest, nil
}

func IsHostInGroup(group int, moderatorId int) bool {
	var res models.ModeratorGroup
	e := DbPool.First(&res, "moderator_group_id LIKE ? AND is_host=1", strconv.Itoa(moderatorId)+","+strconv.Itoa(group))
	if e.Error != nil {
		return false
	}
	return true
}

func GetGroups() ([]models.Group, error) {
	var res []models.Group
	raw := DbPool.Find(&res)
	if errors.Is(raw.Error, gorm.ErrRecordNotFound) {
		return res, nil
	} else if raw.Error != nil {
		return []models.Group{}, raw.Error
	}
	return res, nil
}

func EditContest(contestId int, newContest models.BasicContest) error {
	var existing models.Contest
	res := DbPool.First(&existing, contestId)
	if res.Error != nil {
		return res.Error
	}
	existing.BasicContest = newContest
	DbPool.Save(&existing)
	return nil
}

func GetGroupByContest(contestId int) (int, error) {
	var res models.GroupContestId
	raw := DbPool.First(&res, "group_contest LIKE ? AND belongs=1", "%,"+strconv.Itoa(contestId))
	if raw.Error != nil {
		return -1, raw.Error
	}
	id := strings.Split(res.GroupContest, ",")[0]
	idInt, err := strconv.Atoi(id)
	if err != nil {
		return -1, err
	}
	return idInt, nil
}

func RemoveModeratorInGroup(groupId int, moderatorId int) error {
	var existing models.ModeratorGroup
	res := DbPool.First(&existing, "moderator_group_id = ? AND is_host=1", strconv.Itoa(moderatorId)+","+strconv.Itoa(groupId))
	if errors.Is(res.Error, gorm.ErrRecordNotFound) {
		return nil
	} else if res.Error != nil {
		return res.Error
	}
	existing.IsHost = false
	DbPool.Save(&existing)
	return nil
}

func EditGroup(groupId int, newGroup models.BasicGroup) error {
	var existing models.Group
	res := DbPool.First(&existing, groupId)
	if res.Error != nil {
		return res.Error
	}
	existing.BasicGroup = newGroup
	DbPool.Save(&existing)
	return nil
}
