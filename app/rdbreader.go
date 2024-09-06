package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"strconv"
	"strings"
)

func LoadFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}

	defer func() {
		f.Close()
	}()

	hexReader := hex.NewDecoder(f)
	reader := bufio.NewReader(hexReader)
	metadataByte, _ := RDBHexStringToByte(METADATA_START)
	dbSubsectionByte, _ := RDBHexStringToByte(DB_SUBSECTION_START)
	// hashTableByte, _ := RDBHexStringToByte(HASH_TABLE_START)

	magicStringBytes, err := reader.ReadBytes(metadataByte)
	if err == io.EOF {
		fmt.Println("EOF here", magicStringBytes)
	}
	if err != nil {
		return err
	}

	fmt.Println("header:\n", string(magicStringBytes))

	metadataBytes, err := reader.ReadBytes(dbSubsectionByte)
	if err != nil {
		return err
	}

	lastKey := ""
	nextInsert := "key"
	metadataMap := map[string]interface{}{}
	for len(metadataBytes) > 0 {
		var value interface{}
		attrType, bitRange := decodeByte(metadataBytes[0])
		if attrType == "" {
			return fmt.Errorf("failed to decode byte %v", metadataBytes[0])
		}

		fmt.Println("next type: ", attrType)
		// byteBinaryString := fmt.Sprint("%b", metadataBytes[0])
		ignoreBits, useBits := bitRange[0], bitRange[1]
		useBytes := (ignoreBits + useBits) / 8
		sizeBinaryBits := bytesToBinaryString(metadataBytes[:useBytes])

		valueSizeUpperIndex := ignoreBits + useBits + 1
		fmt.Println("ignore and use bits:", ignoreBits, useBits, sizeBinaryBits)

		if attrType == "string" {
			sizeBinaryBits := bytesToBinaryString(metadataBytes[ignoreBits:valueSizeUpperIndex])
			fmt.Println("number of bits for string:", len(sizeBinaryBits))
			valueSize, err := strconv.ParseInt(sizeBinaryBits, 2, useBits)
			if err != nil {
				fmt.Printf("error parsing valueSize for string %s, %v\n", sizeBinaryBits, err)
				break
			}
			value = string(metadataBytes[valueSizeUpperIndex : valueSize+1])
			fmt.Println("string value: ", value)

		}
		if attrType == "int" {
			valueBytes := metadataBytes[valueSizeUpperIndex : (useBits/8)+1]
			if useBits > 8 {
				slices.Reverse(valueBytes)
			}
			fmt.Println("integer value bytes: ", valueBytes)
			value, err = binary.ReadVarint(bytes.NewReader(valueBytes))
			if err != nil {
				fmt.Println("failed to decode int bytes", err)
				break
			}

			fmt.Println("int value: ", value)
			// sizeBinaryBits = strings. // invert string

			/* sizeBinaryBits := bytesToBinaryString(metadataBytes[ignoreBits:valueSizeUpperIndex])
			fmt.Println("number of bits for int:", len(sizeBinaryBits))
			valueSize, err := strconv.ParseInt(sizeBinaryBits, 2, useBits)
			if err != nil {
				fmt.Println("error parsing valueSize for int", sizeBinaryBits)
			} */
		}

		if nextInsert == "key" {
			nextInsert = "value"
			lastKey = value.(string)
			metadataMap[lastKey] = nil
		} else {
			nextInsert = "key"
			metadataMap[lastKey] = value
		}

		metadataBytes = metadataBytes[valueSizeUpperIndex:]

		// TODO: left here

	}

	fmt.Println("metadata:\n", metadataMap)

	return nil
}

func GetRDBKeys(key string) string {
	return ""
}

func RDBHexStringToByte(hexString string) (byte, error) {
	bytes, err := hex.DecodeString(hexString)

	if err != nil {
		return 0, err
	}

	if len(bytes) != 1 {
		return 0, errors.New("decoded RDB hex string is longer than one byte")
	}

	return bytes[0], nil
}

func decodeByte(startByte byte) (string, [2]int) {
	switch {
	case startByte <= 0b00111111:
		return "string", [2]int{2, 6} // item 1: ignore bits, item 2: transform bits, (item1 + item2) / 8: advance bits
	case startByte <= 0b01111111:
		return "string", [2]int{2, 14}
	case startByte <= 0b10111111:
		return "string", [2]int{8, 32} // ignore 6 remaining bits of the input size
	/* case startByte <= 0b11111111:
	return decodeString(startByte) */
	case startByte == 0xC0:
		return "int", [2]int{8, 8}
	case startByte == 0xC1:
		return "int", [2]int{8, 16}
	case startByte == 0xC2:
		return "int", [2]int{8, 32}
	default:
		return "", [2]int{0, 0}
	}
}

func bytesToBinaryString(inputBytes []byte) string {
	var builder strings.Builder
	for _, b := range inputBytes {
		builder.WriteString(fmt.Sprintf("%08b", b))
	}
	return builder.String()
}
