const express = require("express");
const router = express.Router();

router.get("/", (req, res) => {
  res.send("Cube API working!");
});

module.exports = router;
