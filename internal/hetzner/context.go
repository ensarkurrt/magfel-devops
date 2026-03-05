package hetzner

func (c *Client) SetupContext(name string) error {
	// With HCLOUD_TOKEN env var, context is not strictly needed.
	// But we create one for convenience.
	if c.ResourceExists("context", name) {
		return nil
	}

	_, err := c.Run("context", "create", name)
	return err
}
