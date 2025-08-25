const express = require("express");
const router = express.Router();
const { generateToken } = require("../utils/jwt");

// Dummy user data for example
const users = [{ id: 1, username: "admin", password: "admin123" }];

// Login route
router.post("/login", (req, res) => {
  const { username, password } = req.body;

  const user = users.find(
    (u) => u.username === username && u.password === password
  );

  if (!user) {
    return res.status(401).json({ message: "Invalid credentials" });
  }

  const token = generateToken({ id: user.id, username: user.username });

  res.json({ token });
});

module.exports = router;
