package appeal_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/odpf/guardian/appeal"
	"github.com/odpf/guardian/domain"
	"github.com/odpf/guardian/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type HandlerTestSuite struct {
	suite.Suite
	mockAppealService *mocks.AppealService
	handler           *appeal.Handler
	res               *httptest.ResponseRecorder
}

func (s *HandlerTestSuite) Setup() {
	s.mockAppealService = new(mocks.AppealService)
	s.handler = &appeal.Handler{s.mockAppealService}
	s.res = httptest.NewRecorder()
}

func (s *HandlerTestSuite) SetupTest() {
	s.Setup()
}

func (s *HandlerTestSuite) AfterTest() {
	s.mockAppealService.AssertExpectations(s.T())
}

func (s *HandlerTestSuite) TestCreate() {
	s.Run("should return bad request error if received malformed payload", func() {
		s.Setup()
		malformedPayload := "invalid json format..."
		req, _ := http.NewRequest(http.MethodPost, "/", strings.NewReader(malformedPayload))

		expectedStatusCode := http.StatusBadRequest

		s.handler.Create(s.res, req)
		actualStatusCode := s.res.Result().StatusCode

		s.Equal(expectedStatusCode, actualStatusCode)
	})

	s.Run("should return bad request if payload validation returns error", func() {
		testCases := []struct {
			name           string
			invalidPayload string
		}{
			{
				name: "missing email",
				invalidPayload: `{
	"resources": [
		{
			"id": 1
		},
		{
			"id": 2
		}
	]
}`,
			},
			{
				name: "missing resources",
				invalidPayload: `{
	"email": "test@domain.com"
}`,
			},
			{
				name: "empty resources",
				invalidPayload: `{
	"email": "test@domain.com",
	"resources": []
}`,
			},
		}
		for _, tc := range testCases {
			s.Setup()
			req, _ := http.NewRequest(http.MethodPost, "/", strings.NewReader(tc.invalidPayload))

			expectedStatusCode := http.StatusBadRequest

			s.handler.Create(s.res, req)
			actualStatusCode := s.res.Result().StatusCode

			s.Equal(expectedStatusCode, actualStatusCode)
		}
	})

	validPayload := `{
	"email": "test@email.com",
	"resources": [
		{
			"id": 1
		},
		{
			"id": 2
		}
	]
}`

	s.Run("should return error based on the error thrown by appeal service", func() {
		testCases := []struct {
			expectedServiceError error
			expectedStatusCode   int
		}{
			{
				expectedServiceError: errors.New("appeal service error"),
				expectedStatusCode:   http.StatusInternalServerError,
			},
		}

		for _, tc := range testCases {
			s.Setup()
			req, _ := http.NewRequest(http.MethodPost, "/", strings.NewReader(validPayload))

			s.mockAppealService.On("Create", mock.Anything, mock.Anything).Return(nil, tc.expectedServiceError).Once()

			s.handler.Create(s.res, req)
			actualStatusCode := s.res.Result().StatusCode

			s.Equal(tc.expectedStatusCode, actualStatusCode)
		}
	})

	s.Run("should return newly created appeals on success", func() {
		s.Setup()
		req, _ := http.NewRequest(http.MethodPost, "/", strings.NewReader(validPayload))

		expectedEmail := "test@email.com"
		expectedResourceIDs := []uint{1, 2}
		expectedResponseBody := []*domain.Appeal{
			{
				ID: 1,
			},
			{
				ID: 2,
			},
		}
		s.mockAppealService.On("Create", expectedEmail, expectedResourceIDs).Return(expectedResponseBody, nil).Once()
		expectedStatusCode := http.StatusOK

		s.handler.Create(s.res, req)
		actualStatusCode := s.res.Result().StatusCode
		actualResponseBody := []*domain.Appeal{}
		err := json.NewDecoder(s.res.Body).Decode(&actualResponseBody)
		s.NoError(err)

		s.Equal(expectedStatusCode, actualStatusCode)
		s.Equal(expectedResponseBody, actualResponseBody)
	})
}

func TestHandler(t *testing.T) {
	suite.Run(t, new(HandlerTestSuite))
}