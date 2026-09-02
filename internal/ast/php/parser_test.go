package php

import (
	"context"
	"strings"
	"testing"

	"github.com/nosleepman1/synapse-code/pkg/model"
)

func TestPHPParser(t *testing.T) {
	code := `<?php

namespace App\Http\Controllers;

use App\Models\User;
use App\Services\UserService;
use Illuminate\Http\Request;

interface UserControllerInterface {
    public function index(Request $request);
}

enum AccountStatus: string {
    case ACTIVE = 'active';
    case SUSPENDED = 'suspended';
}

#[Route("/api/v1/users")]
class UserController extends Controller implements UserControllerInterface {
    use HasApiResponse;

    private UserService $userService;

    public function __construct(UserService $userService) {
        $this->userService = $userService;
    }

    #[HttpGet]
    public function index(Request $request) {
        $users = User::where('active', true)->get();
        return $this->success($users);
    }
}
`

	parser := NewParser()
	parsed, err := parser.Parse(context.Background(), "UserController.php", "UserController.php", []byte(code))
	if err != nil {
		t.Fatalf("failed to parse PHP code: %v", err)
	}

	if len(parsed.Imports) < 3 {
		t.Errorf("expected at least 3 imports, got %v", parsed.Imports)
	}

	hasInterface := false
	hasEnum := false
	hasClass := false
	hasCtor := false
	hasIndexMethod := false

	for _, s := range parsed.Symbols {
		if s.Name == "UserControllerInterface" && s.Kind == model.KindInterface {
			hasInterface = true
		}
		if s.Name == "AccountStatus" && s.Kind == model.KindEnum {
			hasEnum = true
		}
		if s.Name == "UserController" && s.Kind == model.KindClass {
			hasClass = true
			if !strings.Contains(s.Signature, "#[Route(") {
				t.Errorf("expected Route attribute in signature, got: %s", s.Signature)
			}
			hasTrait := false
			for _, impl := range s.Implements {
				if impl == "HasApiResponse" {
					hasTrait = true
				}
			}
			if !hasTrait {
				t.Errorf("expected HasApiResponse trait to be captured in implements, got: %v", s.Implements)
			}
		}
		if s.Name == "__construct" && s.Kind == model.KindMethod {
			hasCtor = true
		}
		if s.Name == "index" && s.Kind == model.KindMethod && s.QualifiedName == "UserController::index" {
			hasIndexMethod = true
			if !strings.Contains(s.Signature, "#[HttpGet]") {
				t.Errorf("expected #[HttpGet] attribute in method signature, got: %s", s.Signature)
			}
		}
	}

	if !hasInterface {
		t.Errorf("expected UserControllerInterface interface")
	}
	if !hasEnum {
		t.Errorf("expected AccountStatus enum")
	}
	if !hasClass {
		t.Errorf("expected UserController class")
	}
	if !hasCtor {
		t.Errorf("expected __construct method")
	}
	if !hasIndexMethod {
		t.Errorf("expected index method")
	}
}
