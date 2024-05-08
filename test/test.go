package main

import (
	"log"
	rtl_sdr_mod "rtl_sdr_mod/rtl_sdr"
)

func main() {
	log.Println("*")
	rtl_sdr, err := rtl_sdr_mod.Init("rtl_sdr")
	if err != nil {
		log.Println(err)
		return
	}

	defer func() {
		err := rtl_sdr.Close()
		if err != nil {
			log.Println(err)
		}
	}()

	freq := 433920000
	samp := 1000000
	rf_gain := 9
	buf_size := 262144

	bytes := []byte{}
	for i := 0; ; i++ {
		b, err := rtl_sdr.GetSamplesAsBytes(uint(freq), uint(samp), uint(rf_gain), uint(buf_size))
		if err == rtl_sdr_mod.SampleBytesRetrievalInProgress {
			continue
		}
		if err != nil {
			log.Println(err)
			break
		}

		bytes = append(bytes, b...)
		if i == 9 {
			break
		}
	}
	log.Println(len(bytes))
	log.Println("**")

}
