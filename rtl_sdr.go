package rtl_sdr_mod

import (
	"errors"
	"log"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/go-cmd/cmd"
	"github.com/trailofbits/go-mutexasserts"
)

type RtlSdr struct {
	binary_path             string
	samples_temp_file       *os.File
	samples_temp_file_mutex sync.Mutex
	command                 *cmd.Cmd
	bytes_read              uint
}

var (
	SampleBytesRetrievalTimeout    = errors.New("Sample bytes retrieval timeout")
	SampleBytesRetrievalInProgress = errors.New("Sample bytes retrieval in progress")
)

func Test() string {
	return "TEST"
}

func Init(rtl_sdr_binary_path string) (*RtlSdr, error) {

	rtl_sdr := new(RtlSdr)
	rtl_sdr.binary_path = rtl_sdr_binary_path

	if _, err := os.Stat(rtl_sdr.binary_path); err != nil {
		return nil, err
	}

	samples_temp_file, err := os.CreateTemp("", "")
	if err != nil {
		return nil, err
	}

	rtl_sdr.samples_temp_file = samples_temp_file

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go onCtrlC(c, rtl_sdr)

	return rtl_sdr, nil
}

func onCtrlC(c chan os.Signal, rtl_sdr *RtlSdr) {
	<-c
	rtl_sdr.Close()
	os.Exit(1)
}

func (rtl_sdr *RtlSdr) Close() error {
	if rtl_sdr.command != nil && rtl_sdr.command.Status().Runtime != 0 {

		pid := rtl_sdr.command.Status().PID
		process, err := os.FindProcess(pid)
		if err != nil {
			log.Println(err)
		}

		err = rtl_sdr.command.Stop()
		if err != nil {
			log.Println(err)
		}

		_, err = process.Wait()
		if err != nil {
			log.Println(err)
		}
	}

	for {
		if !mutexasserts.MutexLocked(&rtl_sdr.samples_temp_file_mutex) {
			rtl_sdr.samples_temp_file_mutex.TryLock() //on ctrl c, calls may be made to GetSamplesAsBytes - this stops that
			break
		}
	}

	err := rtl_sdr.samples_temp_file.Close()
	if err != nil {
		log.Fatal(err)
	}
	deleteFile(rtl_sdr.samples_temp_file.Name())
	return nil
}

func (rtl_sdr *RtlSdr) GetSamplesAsBytes(frequency_hz uint, sample_rate_hz uint, rf_gain uint, buffer_size_bytes uint) ([]byte, error) {
	if mutexasserts.MutexLocked(&rtl_sdr.samples_temp_file_mutex) {
		return nil, SampleBytesRetrievalInProgress
	}
	mutex_locked := rtl_sdr.samples_temp_file_mutex.TryLock()
	if !mutex_locked {
		return nil, SampleBytesRetrievalInProgress
	}
	defer rtl_sdr.samples_temp_file_mutex.Unlock()
	bytes := make([]byte, 0)

	if rtl_sdr.command == nil {
		rtl_sdr.command = cmd.NewCmd(rtl_sdr.binary_path, "-f", strconv.Itoa(int(frequency_hz)), "-s", strconv.Itoa(int(sample_rate_hz)), "-g", strconv.Itoa(int(rf_gain)), rtl_sdr.samples_temp_file.Name())
		rtl_sdr.command.Start()
	}

	comparative_time_stamp := time.Now().UnixMilli()
	for {
		time_stamp := time.Now().UnixMilli()
		if (time_stamp - comparative_time_stamp) > 5000 {
			return bytes, SampleBytesRetrievalTimeout
		}
		file_info, err := os.Stat(rtl_sdr.samples_temp_file.Name())
		if err != nil {
			return bytes, err
		}
		if !((file_info.Size() - int64(rtl_sdr.bytes_read)) >= int64(buffer_size_bytes)) {
			continue
		}

		buf := make([]byte, buffer_size_bytes)
		_, err = rtl_sdr.samples_temp_file.ReadAt(buf, int64(rtl_sdr.bytes_read))

		if err != nil {
			return bytes, err
		}

		for _, b := range buf {
			bytes = append(bytes, b)
		}
		break
	}

	rtl_sdr.bytes_read += uint(len(bytes))
	return bytes, nil
}
