const express = require("express");
const router = express.Router();

router.get("/", (req, res) => {
  res.send("KPI API working!");
});

module.exports = router;
