package main

import (
	"fmt"
	"os"
	"sync"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type SerialKey struct {
	gorm.Model
	Key       string `gorm:"index"`
	SerialNum uint
}

type Spoolfile struct {
	gorm.Model
	DFID     uint `gorm:"index"`
	File     string
	Name     string
	Type     uint8 `gorm:"index"`
	Device   string
	Recs     uint
	Priority uint
	Copies   uint
}
type Job struct {
	gorm.Model
	JobID       uint `gorm:"index"`
	StreamedBy  string
	pid         uint
	ppid        uint
	Name        string
	Acct        string
	Group       string
	Intro       time.Time
	Logon       time.Time
	Eoj         time.Time
	OutID       uint   `gorm:"index"`
	InID        uint   `gorm:"index"`
	OutDev      string `gorm:"index"`
	OutPri      uint
	InPri       uint
	CpuQ        string `gorm:"index"`
	CpuTime     time.Duration
	ElapsedTime time.Duration
}

func Create() {
	db, err := gorm.Open(sqlite.Open("spool.db"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	// Migrate the schema
	db.AutoMigrate(&SerialKey{})
	db.AutoMigrate(&Spoolfile{})
	db.AutoMigrate(&Job{})
}

func nextSerial(db *gorm.DB, k string) (uint, error) {
	var l sync.Mutex
	var sk SerialKey
	l.Lock()
	defer l.Unlock()

	tx := db.Begin()
	tx = db.Model(&sk).Where("key = ?", k).First(&sk)

	if tx.Error == gorm.ErrRecordNotFound {
		fmt.Println("Record not found: ", k)
		sk = SerialKey{Key: k, SerialNum: 0}
		db = db.Model(&sk).Create(&sk)
	} else {
		if tx.Error != nil {
			tx.Rollback()
			return 0, fmt.Errorf("%w", tx.Error)
		}
	}
	if db.Error != nil {
		tx.Rollback()
		return 0, fmt.Errorf("%w", tx.Error)
	}

	sk.SerialNum++
	db.Model(&sk).Where("key = ?", k).Update("serial_num", sk.SerialNum)
	tx.Commit()

	return sk.SerialNum, nil
}

func SpoolIn(db *gorm.DB) (Spoolfile, error) {

	spi := Spoolfile{
		DFID:     0,
		Name:     "$STDIN",
		Type:     1,
		Device:   "LP",
		Recs:     0,
		Priority: 8,
		Copies:   1,
	}

	var err error
	spi.DFID, err = nextSerial(db, "I")
	if err != nil {
		fmt.Println(err)
	}

	spi.File = fmt.Sprintf("/spool/in/I%d.spt", spi.DFID)
	result := db.Create(&spi)
	if result.Error != nil {
		return Spoolfile{}, result.Error
	}

	return spi, nil
}

func SpoolOut(db *gorm.DB) (Spoolfile, error) {

	spo := Spoolfile{
		DFID:     0,
		Name:     "$STDLIST",
		Type:     0,
		Device:   "LP",
		Recs:     0,
		Priority: 8,
		Copies:   1,
	}
	var err error
	spo.DFID, err = nextSerial(db, "O")
	if err != nil {
		fmt.Println(err)
	}

	spo.File = fmt.Sprintf("/spool/out/O%d.spt", spo.DFID)
	result := db.Create(&spo)
	if result.Error != nil {
		return Spoolfile{}, result.Error
	}

	return spo, nil
}

func Newjob(j *Job) (*Job, error) {
	db, err := gorm.Open(sqlite.Open("spool.db"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}
	if len(j.Name) < 1 {
		j.Name = "noname"
	}
	if len(j.Acct) < 1 {
		j.Acct = "nobody"
	}
	if len(j.Group) < 1 {
		j.Group = "homeless"
	}
	if j.InPri < 1 {
		j.InPri = 8
	}
	if len(j.CpuQ) < 1 {
		j.CpuQ = "ds"
	}

	j.Intro = time.Now()
	j.ppid = uint(os.Getpid())
	j.CpuTime = 0
	j.ElapsedTime = 0
	var sl, si Spoolfile
	sl, err = SpoolOut(db)
	if err != nil {
		return &Job{}, err
	}

	si, err = SpoolIn(db)
	if err != nil {
		return &Job{}, err
	}

	j.OutID = sl.DFID
	j.InID = si.DFID

	j.JobID, err = nextSerial(db, "J")
	if err != nil {
		return &Job{}, err
	}

	result := db.Create(&j)

	if result.Error != nil {
		return &Job{}, result.Error
	}

	return j, nil
}

func main() {
	Create()
	var job Job
	j, err := Newjob(&job)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Printf("#J%d\n", j.JobID)

}
