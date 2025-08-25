

const jwt = require("jsonwebtoken");

const generateToken = (user) => {
  const payload = {
    id: user.id,
    username: user.username,
    role: user.role,
  };
  const accessToken = jwt.sign(payload, process.env.JWT_SECRET, {
    expiresIn: "1h",
  });
  return accessToken;
};

module.exports = { generateToken };
