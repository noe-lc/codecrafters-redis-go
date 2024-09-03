package main

import (
	"bufio"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
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

	reader := bufio.NewReader(f)
	metadataByte, _ := RDBHexStringToByte(METADATA_START)
	dbSubsectionByte, _ := RDBHexStringToByte(DB_SUBSECTION_START)
	hashTableByte, _ := RDBHexStringToByte(HASH_TABLE_START)

	magicStringBytes, err := reader.ReadBytes(metadataByte)
	if err != nil {
		return err
	}

	fmt.Println(string(magicStringBytes))

	metadataBytes, err := reader.ReadBytes(dbSubsectionByte)
	if err != nil {
		return err
	}

	nextByteIndex := 0
	for nextByteIndex < len(metadataBytes)-1 {
		attrType, bits := decodeByte(metadataBytes[nextByteIndex])
		if attrType == "" {
			return fmt.Errorf("failed to decode byte %v at position %i", metadataBytes[nextByteIndex], nextByteIndex)
		}

		fmt.Println("next type: ", attrType)
		ignoreBits, useBits := bits[0], bits[1]
		nextByteIndex += (ignoreBits + useBits) / 8
		sizeBinaryBits := bytesToBinaryString(metadataBytes[:nextByteIndex])[ignoreBits:]

		// TODO: left here

	}

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
