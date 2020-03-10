package controller

import (
    "net/http"
    "github.com/gin-gonic/gin"
)

type HandlerFunc func(c *gin.Context) error

type Response struct {
    Code    int `json:"code"`
    Msg     string `json:"msg"`
    Request string `json:"request"`
}

func Wrapper(handler HandlerFunc) func(c *gin.Context) {
    return func(c *gin.Context) {
        var err error
        var response *Response
        err = handler(c)
        if err != nil {
            if h, ok := err.(*Response); ok {
                response = h
            } else {
                response = ServerError()
            }
        } else {
            response = OK()
        }
        response.Request = c.Request.Method + " " + c.Request.URL.String()
        c.JSON(response.Code, response)
        return
    }
}

func (resp *Response) Error() string {
    return resp.Msg
}

func newResponse(code int, msg string) * Response {
    return &Response{
        Code: code,
        Msg: msg,
    }
}

func NotFound() *Response {
    return newResponse(http.StatusNotFound, http.StatusText(http.StatusNotFound))
}

func ServerError() *Response {
    return newResponse(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
}

func OK() *Response {
    return newResponse(http.StatusOK, http.StatusText(http.StatusOK))
}

func BadRequest() *Response {
    return newResponse(http.StatusBadRequest, http.StatusText(http.StatusBadRequest))
}

func HandleNotFound(c *gin.Context) {
    handle := NotFound()
    handle.Request = c.Request.Method + " " + c.Request.URL.String()
    c.JSON(handle.Code, handle)
    return
}
