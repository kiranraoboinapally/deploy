const express = require("express");
const router = express.Router();
const {
  getCubes,
  createCube,
  addFactToCube,
  addDimensionToCube
} = require("../controllers/cubesController");

router.get("/", getCubes);
router.post("/", createCube);
router.post("/:cubeId/facts", addFactToCube);
router.post("/:cubeId/dimensions", addDimensionToCube);

module.exports = router;
