const pool = require("../models/db");
const bcrypt = require("bcryptjs");
const { generateToken } = require("../utils/jwt");

// Register Controller
const register = async (req, res) => {
  const { username, email, password, role = "user" } = req.body;

  if (!username || !email || !password) {
    return res.status(400).json({ message: "Please provide all required fields" });
  }

  try {
    // Check if user already exists
    const userExists = await pool.query("SELECT * FROM users WHERE email = $1", [email]);
    if (userExists.rows.length > 0) {
      return res.status(400).json({ message: "User already exists" });
    }

    // Hash password
    const salt = await bcrypt.genSalt(10);
    const hashedPassword = await bcrypt.hash(password, salt);

    // Insert new user
    const result = await pool.query(
      `INSERT INTO users (username, email, password, role)
       VALUES ($1, $2, $3, $4)
       RETURNING id, username, email, role`,
      [username, email, hashedPassword, role]
    );

    const newUser = result.rows[0];
    const token = generateToken(newUser); // Should include user.id, user.email, user.role

    return res.status(201).json({
      user: newUser,
      token,
    });

  } catch (err) {
    console.error("Register Error:", err.message);
    return res.status(500).json({ message: "Server error" });
  }
};

// Login Controller
const login = async (req, res) => {
  const { email, password } = req.body;

  if (!email || !password) {
    return res.status(400).json({ message: "Please provide all required fields" });
  }

  try {
    const result = await pool.query("SELECT * FROM users WHERE email = $1", [email]);
    if (result.rows.length === 0) {
      return res.status(400).json({ message: "Invalid credentials" });
    }

    const user = result.rows[0];
    const isMatch = await bcrypt.compare(password, user.password);

    if (!isMatch) {
      return res.status(400).json({ message: "Invalid credentials" });
    }

    // Do not include password in token or response
    const token = generateToken({
      id: user.id,
      username: user.username,
      email: user.email,
      role: user.role,
    });

    return res.json({
      user: {
        id: user.id,
        username: user.username,
        email: user.email,
        role: user.role,
      },
      token,
    });

  } catch (err) {
    console.error("Login Error:", err.message);
    return res.status(500).json({ message: "Server error" });
  }
};

// Logout Controller
const logout = (req, res) => {
  res.clearCookie("token");
  return res.status(200).json({ message: "Logged out successfully" });
};

module.exports = {
  register,
  login,
  logout,
};
