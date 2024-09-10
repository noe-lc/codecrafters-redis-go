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

type RDBValue interface {
	int | string
}

func LoadFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}

	defer func() {
		f.Close()
	}()

	fileHash := make([]byte, 8)
	hexReader := hex.NewDecoder(f)
	reader := bufio.NewReader(hexReader)
	metadataByte, _ := RDBHexStringToByte(RDB_METADATA_START)
	dbSubsectionByte, _ := RDBHexStringToByte(RDB_DB_SUBSECTION_START)
	// hashTableByte, _ := RDBHexStringToByte(RDB_HASH_TABLE_START)
	endOfFileByte, _ := RDBHexStringToByte(RDB_END_OF_FILE)

	magicStringBytes, err := reader.ReadBytes(metadataByte)
	if err != nil {
		if err == io.EOF {
			fmt.Println("EOF here", magicStringBytes)
			return err
		}
		return err
	}

	// TODO: check whether or not the RDB file is empty
	bytesUntilEOF, err := reader.ReadBytes(endOfFileByte)

	metadataSectionBytes, err := reader.ReadBytes(dbSubsectionByte)
	if err != nil {
		return err
	}

	lastKey := ""
	nextInsert := "key"
	metadataMap := map[string]interface{}{}
	metadataSectionBytes = metadataSectionBytes[:len(metadataSectionBytes)-1]
	metadataReader := bufio.NewReader(bytes.NewReader(metadataSectionBytes))

	for {
		sectionBytes, err := metadataReader.ReadBytes(metadataByte)
		if err != nil {
			if err != io.EOF {
				return fmt.Errorf("error reading metadata: %v", err)
			}
			if len(sectionBytes) == 0 {
				fmt.Println("finished reading metadata")
				break
			}
			break // end of metadata read
		}

		sectionBytes = sectionBytes[:len(sectionBytes)-1] // remove last byte (the subsection separator)

		for {
			if len(sectionBytes) == 0 {
				break
			}
			value, bitRange, err := decodeNextValue(sectionBytes)
			if err != nil {
				return err
			}
			ignoreBits, useBits := bitRange[0], bitRange[1]
			useBytes := (ignoreBits + useBits) / 8 // calculate total bytes for read

			switch v := value.(type) {
			case int:
				sectionBytes = sectionBytes[useBytes:]
			case string:
				sectionBytes = sectionBytes[useBytes+len(v):]
			default:
				break
			}

			if nextInsert == "key" {
				nextInsert = "value"
				lastKey = value.(string)
				metadataMap[lastKey] = nil
			} else {
				nextInsert = "key"
				metadataMap[lastKey] = value
			}

		}

		fmt.Println("metadata:\n", metadataMap)
	}

	dbIndex, err := reader.Peek(1)
	if err != nil {
		return err
	}
	// empty rdb file
	if dbIndex[0] == 0xC0 {
		_, err := reader.ReadBytes(endOfFileByte)
		if err != nil {
			return err
		}
		bytesRead, _ := reader.Read(fileHash)
		fmt.Println("Finished reading file. Hash is: ", fileHash[:bytesRead])
		return nil
	}

	// non-empty rdb file

	for {
		break
	}

	return nil
}

// decodeString returns the
func decodeString(input []byte, bitRange [2]int) (string, error) {
	ignoreBits, useBits := bitRange[0], bitRange[1]
	useBytes := (ignoreBits + useBits) / 8                                 // calculate total bytes to take from input
	binaryStringSize := bytesToBinaryString(input[:useBytes])[ignoreBits:] // take only required bits
	stringLength, err := strconv.ParseInt(binaryStringSize, 2, useBits)
	if err != nil {
		return "", err
	}
	return string(input[useBytes:][:stringLength]), nil
}

func decodeInt(input []byte, bitRange [2]int) (int, error) {
	ignoreBits, useBits := bitRange[0], bitRange[1]
	useBytes := (ignoreBits + useBits) / 8 // calculate total bytes to take from input
	binaryBitSize := bytesToBinaryString(input[:useBytes])[ignoreBits:]
	if useBits > 8 {
		sizeBinaryBitSlice := strings.Split(binaryBitSize, "") // TODO: check how to reverse an actual string
		slices.Reverse(sizeBinaryBitSlice)
		binaryBitSize = strings.Join(sizeBinaryBitSlice, "")
	}
	fmt.Println("integer value bits: ", binaryBitSize)
	integer, err := binary.ReadVarint(bytes.NewReader([]byte(binaryBitSize)))
	if err != nil {
		fmt.Println("failed to decode int bytes", err)
		return 0, err
	}
	return int(integer), nil
}

// decodeBytes returns the RDBType and an 2-item array with the exact bits to ignore and use.
// If there are no matches, ti returns an empty string and [2]int{0,0}
func decodeNextValue(inputBytes []byte) (interface{}, [2]int, error) {
	var err error
	var decodedValue interface{}
	var bitRange [2]int = [2]int{0, 0}
	startByte := inputBytes[0]

	switch {
	case startByte <= 0b00111111:
		bitRange = [2]int{2, 6}
		decodedValue, err = decodeString(inputBytes, bitRange) // item 1: ignore bits, item 2: transform bits, (item1 + item2) / 8: advance bits
	case startByte <= 0b01111111:
		bitRange = [2]int{2, 14}
		decodedValue, err = decodeString(inputBytes, bitRange)
	case startByte <= 0b10111111:
		bitRange = [2]int{8, 32} // ignore 6 remaining bits of the input size
		decodedValue, err = decodeString(inputBytes, bitRange)
	/* case startByte <= 0b11111111:
	return decodeString(startByte) */
	case startByte == 0xC0:
		bitRange = [2]int{8, 8}
		decodedValue, err = decodeInt(inputBytes, bitRange)
	case startByte == 0xC1:
		bitRange = [2]int{8, 16}
		decodedValue, err = decodeInt(inputBytes, bitRange)
	case startByte == 0xC2:
		bitRange = [2]int{8, 32}
		decodedValue, err = decodeInt(inputBytes, bitRange)
	default:
		err = fmt.Errorf("unable to decode byte %d", startByte)
	}

	if err != nil {
		return decodedValue, bitRange, err
	}
	return decodedValue, bitRange, nil
}

func bytesToBinaryString(inputBytes []byte) string {
	var builder strings.Builder
	for _, b := range inputBytes {
		builder.WriteString(fmt.Sprintf("%08b", b))
	}
	return builder.String()
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

func GetRDBKeys(filePath, key string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}

	hexReader := hex.NewDecoder(f)
	reader := bufio.NewReader(hexReader)
	dbSubsectionByte, _ := RDBHexStringToByte(RDB_DB_SUBSECTION_START)
	stringKeyByte, _ := RDBHexStringToByte(RDB_STRING_KEY)
	hashTableByte, _ := RDBHexStringToByte(RDB_HASH_TABLE_START)

	// advance until keys
	_, err = reader.ReadBytes(dbSubsectionByte)
	if err != nil {
		return "", err
	}
	_, err = reader.ReadBytes(hashTableByte)
	if err != nil {
		return "", err
	}
	_, err = reader.Discard(4)
	if err != nil {
		return "", err
	}
	_, err = reader.ReadBytes(stringKeyByte)
	if err != nil {
		return "", err
	}

	for {
		_, err = reader.ReadBytes(stringKeyByte)
		if err != nil {
			return "", err
		}
		break
		//n _, err := reader.Peek()
	}

	return "", nil
}
