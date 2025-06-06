-- Database initialization for Go Kanban project
-- Based on your existing table structures

-- Create users table
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    username CHARACTER VARYING(255) UNIQUE NOT NULL,
    password CHARACTER VARYING(255) NOT NULL,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create projects table
CREATE TABLE IF NOT EXISTS projects (
    id SERIAL PRIMARY KEY,
    name CHARACTER VARYING(255) NOT NULL,
    repo_url CHARACTER VARYING(500),
    site_url CHARACTER VARYING(500),
    description TEXT,
    dependencies TEXT[],
    dev_dependencies TEXT[],
    status CHARACTER VARYING(50) DEFAULT 'active',
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_projects_user_id ON projects(user_id);
CREATE INDEX IF NOT EXISTS idx_projects_status ON projects(status);
CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);

-- Insert a default admin user (optional)
INSERT INTO users (username, password, is_active) VALUES 
('admin', '$2a$10$dummy.hash.replace.with.real.hash', true)
ON CONFLICT (username) DO NOTHING;