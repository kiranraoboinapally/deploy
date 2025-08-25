const express = require("express");
const router = express.Router();
const { getDimensions, createDimension } = require("../controllers/dimensionsController");

router.get("/", getDimensions);
router.post("/", createDimension);

module.exports = router;
