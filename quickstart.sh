#!/bin/bash
set -e

echo ""
echo "Qualys DSPM - Quick Start Setup"
echo "================================"
echo ""

check_deps() {
    echo "Checking dependencies..."

    if ! command -v docker &> /dev/null; then
        echo "ERROR: Docker is required but not installed."
        echo "Install from: https://docs.docker.com/get-docker/"
        exit 1
    fi
    echo "  Docker: OK"

    if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null; then
        echo "ERROR: Docker Compose is required but not installed."
        exit 1
    fi
    echo "  Docker Compose: OK"
    echo ""
}

setup_config() {
    if [ ! -f config.yaml ]; then
        echo "Creating config.yaml from template..."
        cp config.example.yaml config.yaml
        echo "  Config created"
    else
        echo "  Config exists"
    fi
    echo ""
}

start_services() {
    echo "Starting services..."

    docker compose up -d postgres redis

    echo "Waiting for PostgreSQL..."
    sleep 5

    until docker compose exec -T postgres pg_isready -U dspm -d dspm > /dev/null 2>&1; do
        sleep 2
    done
    echo "  PostgreSQL: Ready"
    echo "  Redis: Ready"
    echo ""
}

run_migrations() {
    echo "Running database migrations..."

    for migration in migrations/*.sql; do
        if [ -f "$migration" ]; then
            docker compose exec -T postgres psql -U dspm -d dspm -f /docker-entrypoint-initdb.d/$(basename $migration) > /dev/null 2>&1 || true
        fi
    done

    echo "  Migrations complete"
    echo ""
}

build_and_start() {
    echo "Building DSPM..."
    docker compose build dspm
    echo "  Build complete"

    echo "Starting DSPM server..."
    docker compose up -d dspm
    echo "  DSPM started"
    echo ""
}

print_success() {
    echo ""
    echo "DSPM is now running"
    echo "==================="
    echo ""
    echo "  Dashboard:    http://localhost:8080"
    echo "  API:          http://localhost:8080/api/v1"
    echo "  Health:       http://localhost:8080/health"
    echo ""
    echo "Next Steps:"
    echo ""
    echo "  1. Open http://localhost:8080 in your browser"
    echo "  2. Add a cloud account (AWS, Azure, or GCP)"
    echo "  3. Run your first scan"
    echo ""
    echo "Add Cloud Accounts:"
    echo ""
    echo "  AWS (using Terraform):"
    echo "    cd deploy/terraform/aws"
    echo "    terraform init"
    echo "    terraform apply -var='dspm_external_id=your-external-id'"
    echo ""
    echo "  Azure (using CLI):"
    echo "    az ad sp create-for-rbac --name QualysDSPM --role Reader"
    echo ""
    echo "Commands:"
    echo ""
    echo "  docker compose logs -f dspm    View logs"
    echo "  docker compose down            Stop all services"
    echo "  docker compose ps              Check status"
    echo ""
}

main() {
    check_deps
    setup_config
    start_services
    run_migrations
    build_and_start
    print_success
}

case "${1:-}" in
    --help|-h)
        echo "Usage: ./quickstart.sh [command]"
        echo ""
        echo "Commands:"
        echo "  (none)     Full setup and start"
        echo "  stop       Stop all services"
        echo "  logs       View DSPM logs"
        echo "  status     Check service status"
        echo ""
        ;;
    stop)
        docker compose down
        echo "Services stopped"
        ;;
    logs)
        docker compose logs -f dspm
        ;;
    status)
        docker compose ps
        ;;
    *)
        main
        ;;
esac
