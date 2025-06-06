-- Create the update function
CREATE OR REPLACE FUNCTION update_updated_at_column() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$;

-- Create users table
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(255) NOT NULL UNIQUE,
    password VARCHAR(255) NOT NULL,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

-- Create projects table
CREATE TABLE IF NOT EXISTS projects (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    repo_url VARCHAR(500),
    site_url VARCHAR(500),
    description TEXT,
    dependencies TEXT[],
    dev_dependencies TEXT[],
    status VARCHAR(50) DEFAULT 'active',
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
CREATE INDEX IF NOT EXISTS idx_projects_user_id ON projects(user_id);
CREATE INDEX IF NOT EXISTS idx_projects_status ON projects(status);
CREATE INDEX IF NOT EXISTS idx_projects_created_at ON projects(created_at);

-- Create triggers
DROP TRIGGER IF EXISTS update_users_updated_at ON users;
CREATE TRIGGER update_users_updated_at 
    BEFORE UPDATE ON users 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_projects_updated_at ON projects;
CREATE TRIGGER update_projects_updated_at 
    BEFORE UPDATE ON projects 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Insert sample data
INSERT INTO users (username, password, is_active) VALUES 
('Daniyar', '$2a$10$.2jTS3FdbLRhYVGDL2fD.u2Ux4eMcao5T3uWFFNby3n7y61jKa6kW', true),
('Alim', '$2a$10$5/C6.c/QMbM7mA5lRo9mU.pRinpZU5jxgUzLaDxThVMr.kuqa5Lpa', true),
('Asset', '$2a$10$92fQE1OdGWP2W.adb.rvNOBXjh1ikFIqmDwf9/Dx0pE8BzGdIvz.W', true),
('Asset2', '$2a$10$Xi9jXbVjDSdpVHbM1lefVuQNys.9E2ma0ACHWEu.a5KgVqZWMVBne', true)
ON CONFLICT (username) DO NOTHING;

INSERT INTO projects (name, repo_url, site_url, dependencies, dev_dependencies, status, user_id) VALUES 
('Test', 'https://www.youtube.com/watch?v=7cICKi__1_E', 'https://www.youtube.com/watch?v=7cICKi__1_E', 
 ARRAY['react','react-dom'], ARRAY['@types/react-dom','@types/react'], 'active', 4),
('Test', 'https://www.youtube.com/watch?v=7cICKi__1_E', 'https://www.youtube.com/watch?v=7cICKi__1_E', 
 ARRAY['react','react-dom'], ARRAY['@types/react-dom','@types/react'], 'developing', 4),
('Test', 'https://www.youtube.com/watch?v=7cICKi__1_E', 'https://www.youtube.com/watch?v=7cICKi__1_E', 
 ARRAY['react','react-dom'], ARRAY['@types/react-dom','@types/react'], 'active', 4)
ON CONFLICT DO NOTHING;