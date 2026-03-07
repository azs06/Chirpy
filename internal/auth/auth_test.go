package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestValidateJWT(t *testing.T) {
	userId := uuid.New()
	validToken, _ := MakeJWT(userId, "test", time.Hour)
	expiredToken, _ := MakeJWT(userId, "test", -time.Hour)
	tests := []struct {
		name    string
		token   string
		secret  string
		wantErr bool
	}{
		{"valid token", validToken, "test", false},
		{"wrong secret", validToken, "wrong", true},
		{"garbage string", "abc.def.ghi", "secret", true},
		{"expired token", expiredToken, "test", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ValidateJWT(tt.token, tt.secret)
			if (err != nil) != tt.wantErr {
				t.Errorf("got err=%v, wantErr=%v", err, tt.wantErr)
			}
		})
	}
}

func TestMakeAndValidateJWT(t *testing.T) {
	userId := uuid.New()
	validToken, err := MakeJWT(userId, "secret", time.Hour)
	if err != nil {
		t.Fatalf("MakeJWT failed: %v", err)
	}
	userUUID, err := ValidateJWT(validToken, "secret")
	if err != nil {
		t.Fatalf("ValidateJWT failed: %v", err)
	}

	if userId != userUUID {
		t.Errorf("got userID=%v, want=%v", userUUID, userId)
	}

}
