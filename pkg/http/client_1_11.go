// +build !go1.12

package http

func (c *Client) Close() {
	// c.httpClient.CloseIdleConnections()
}
