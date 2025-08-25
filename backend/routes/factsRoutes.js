const express = require("express");
const router = express.Router();
const {
  getFacts,
  createFact,
} = require("../controllers/factsController");

router.get("/", getFacts);       // Get all facts
router.post("/", createFact);    // Admin: Create a new fact

module.exports = router;
