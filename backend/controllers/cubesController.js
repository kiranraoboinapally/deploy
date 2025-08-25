const pool = require("../models/db");

exports.getCubes = async (req, res) => {
  const result = await pool.query("SELECT * FROM cubes ORDER BY id");
  res.json(result.rows);
};

exports.createCube = async (req, res) => {
  const { name, description } = req.body;
  const result = await pool.query(
    `INSERT INTO cubes (name, description) VALUES ($1, $2) RETURNING *`,
    [name, description]
  );
  res.status(201).json(result.rows[0]);
};

exports.addFactToCube = async (req, res) => {
  const { cubeId } = req.params;
  const { fact_id } = req.body;
  const result = await pool.query(
    `INSERT INTO cube_facts (cube_id, fact_id) VALUES ($1, $2) RETURNING *`,
    [cubeId, fact_id]
  );
  res.status(201).json(result.rows[0]);
};

exports.addDimensionToCube = async (req, res) => {
  const { cubeId } = req.params;
  const { dimension_id } = req.body;
  const result = await pool.query(
    `INSERT INTO cube_dimensions (cube_id, dimension_id) VALUES ($1, $2) RETURNING *`,
    [cubeId, dimension_id]
  );
  res.status(201).json(result.rows[0]);
};
