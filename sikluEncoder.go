package gosiklu

import (
	"strings"
)

var stdBase64Chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"

func base64Encode3To4(inputStr string) string {
	var outputStr strings.Builder

	temp := (inputStr[0] >> 2) & 0x3F
	outputStr.WriteByte(stdBase64Chars[temp])

	temp = (((inputStr[1] >> 4) & 0x0F) + ((inputStr[0] << 4) & 0x30)) & 0x3F
	outputStr.WriteByte(stdBase64Chars[temp])

	temp = (((inputStr[1] << 2) & 0x3C) + ((inputStr[2] >> 6) & 0x03)) & 0x3F
	outputStr.WriteByte(stdBase64Chars[temp])

	temp = inputStr[2] & 0x3F
	outputStr.WriteByte(stdBase64Chars[temp])

	return outputStr.String()
}

func passwordEncode(password string) string {
	originalPasswordLength := len(password)
	var encodedPassword strings.Builder

	// Add padding if necessary
	if len(password)%3 == 1 {
		password += "=="
	} else if len(password)%3 == 2 {
		password += "="
	}

	for ix := 0; ix < len(password); ix += 3 {
		inputStr := password[ix : ix+3]
		outputStr := base64Encode3To4(inputStr)
		encodedPassword.WriteString(outputStr)
	}

	if originalPasswordLength%3 == 1 {
		encodedPassword.WriteString("2")
	} else if originalPasswordLength%3 == 2 {
		encodedPassword.WriteString("1")
	} else {
		encodedPassword.WriteString("0")
	}

	return encodedPassword.String()
}
