const express = require("express");
const router = express.Router();

// Placeholder routes for facts, dimensions, cubes metadata
router.get("/", (req, res) => {
  console.log("Metadata route hit");
  res.send("Metadata API is working!");
});

module.exports = router;
