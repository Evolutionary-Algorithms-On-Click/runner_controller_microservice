package modules

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
)

type BOBound struct {
	Min float64 `json:"min"`
	Max float64 `json:"max"`
}

type BOInitialDesign struct {
	Strategy  string  `json:"strategy"`
	LHSType   *string `json:"lhs_type"`
	Criterion *string `json:"criterion"`
}

type BOParams struct {
	InitialPoints int      `json:"initialPoints"`
	Iterations    int      `json:"iterations"`
	Verbose       bool     `json:"verbose"`
	Xi            *float64 `json:"xi"`
	Kappa         *float64 `json:"kappa"`
	RandomSeed    *int     `json:"randomSeed"`
}

type BOCustomFunction struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type BO struct {
	AlgorithmType  string            `json:"algorithm_type"`
	Direction      string            `json:"direction"`
	Objective      string            `json:"objective"`
	CustomFunction *BOCustomFunction `json:"custom_function"`
	Surrogate      string            `json:"surrogate"`
	Acquisition    string            `json:"acquisition"`
	Kernel         *string           `json:"kernel"`
	Bounds         []BOBound         `json:"bounds"`
	InitialDesign  BOInitialDesign   `json:"initial_design"`
	Params         BOParams          `json:"params"`
}

func BOFromJSON(jsonData map[string]any) (*BO, error) {
	bo := &BO{}
	jsonDataBytes, err := json.Marshal(jsonData)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(jsonDataBytes, bo); err != nil {
		return nil, err
	}
	return bo, nil
}

func (bo *BO) validate() error {
	// Validate algorithm type
	if bo.AlgorithmType != "standard_bo" {
		return fmt.Errorf("invalid algorithm type: %s", bo.AlgorithmType)
	}

	// Validate direction
	if !slices.Contains([]string{"minimize", "maximize"}, bo.Direction) {
		return fmt.Errorf("invalid direction: %s", bo.Direction)
	}

	// Validate objective
	validObjectives := []string{
		"sphere", "rosenbrock", "ackley", "rastrigin", "schwefel",
		"griewank", "levy", "michalewicz", "beale", "booth",
		"matyas", "branin", "six_hump_camel", "himmelblau",
		"goldstein_price", "easom", "custom",
	}
	if !slices.Contains(validObjectives, bo.Objective) {
		return fmt.Errorf("invalid objective function: %s", bo.Objective)
	}

	// Validate custom function if objective is custom
	if bo.Objective == "custom" {
		if bo.CustomFunction == nil {
			return fmt.Errorf("custom function required when objective is 'custom'")
		}
		if bo.CustomFunction.Code == "" || bo.CustomFunction.Name == "" {
			return fmt.Errorf("custom function code and name are required")
		}
	}

	// Validate surrogate
	if !slices.Contains([]string{"gp", "rf"}, bo.Surrogate) {
		return fmt.Errorf("invalid surrogate model: %s", bo.Surrogate)
	}

	// Validate kernel for GP
	if bo.Surrogate == "gp" {
		if bo.Kernel == nil {
			return fmt.Errorf("kernel required for GP surrogate")
		}
		validKernels := []string{
			"rbf", "matern_2.5", "matern_1.5", "rational_quadratic",
			"exp_sine_squared", "dot_product", "white", "constant",
		}
		if !slices.Contains(validKernels, *bo.Kernel) {
			return fmt.Errorf("invalid kernel: %s", *bo.Kernel)
		}
	}

	// Validate acquisition
	if !slices.Contains([]string{"ei", "pi", "lcb"}, bo.Acquisition) {
		return fmt.Errorf("invalid acquisition function: %s", bo.Acquisition)
	}

	// Validate bounds
	if len(bo.Bounds) == 0 {
		return fmt.Errorf("at least one dimension bound is required")
	}
	for i, bound := range bo.Bounds {
		if bound.Min >= bound.Max {
			return fmt.Errorf("invalid bounds for dimension %d: min=%f, max=%f", i, bound.Min, bound.Max)
		}
	}

	// Validate initial design
	validStrategies := []string{"random", "lhs", "sobol", "halton", "hammersley", "grid"}
	if !slices.Contains(validStrategies, bo.InitialDesign.Strategy) {
		return fmt.Errorf("invalid initial design strategy: %s", bo.InitialDesign.Strategy)
	}

	// Validate LHS options
	if bo.InitialDesign.Strategy == "lhs" {
		if bo.InitialDesign.LHSType != nil {
			if !slices.Contains([]string{"classic", "centered"}, *bo.InitialDesign.LHSType) {
				return fmt.Errorf("invalid LHS type: %s", *bo.InitialDesign.LHSType)
			}
		}
		if bo.InitialDesign.Criterion != nil {
			if !slices.Contains([]string{"maximin", "correlation", "ratio"}, *bo.InitialDesign.Criterion) {
				return fmt.Errorf("invalid LHS criterion: %s", *bo.InitialDesign.Criterion)
			}
		}
	}

	// Validate params
	if bo.Params.InitialPoints < 1 {
		return fmt.Errorf("initial points must be at least 1")
	}
	if bo.Params.Iterations < 1 {
		return fmt.Errorf("iterations must be at least 1")
	}

	return nil
}

func (bo *BO) imports() string {
	return strings.Join([]string{
		"#!/usr/bin/env python3",
		"import os",
		"import uuid",
		"import math",
		"import time",
		"import numpy as np",
		"import matplotlib.pyplot as plt",
		"import matplotlib.animation as animation",
		"from skopt import gp_minimize, forest_minimize",
		"from skopt.space import Real",
		"from skopt.space import Space",
		"from skopt.sampler import Sobol, Lhs, Halton, Hammersly, Grid",
		"from skopt.learning import GaussianProcessRegressor",
		"from skopt.learning.gaussian_process.kernels import (",
		"    RBF, Matern, RationalQuadratic, DotProduct, WhiteKernel,",
		"    ExpSineSquared, ConstantKernel",
		")",
		"from landscapes.single_objective import (",
		"    sphere, rosenbrock, ackley, rastrigin, schwefel, griewank,",
		"    levi_n13, michalewicz, beale, booth, matyas,",
		"    branin, camel_hump_6, himmelblau, goldstein_price, easom",
		")",
	}, "\n")
}

func (bo *BO) benchmarkFunctions() string {
	return strings.Join([]string{
		"\n# Benchmark Functions (using landscapes library)",
		"# Wrapper functions for naming consistency with our API",
		"def levy(x): return levi_n13(x)",
		"def six_hump_camel(x): return camel_hump_6(x)",
		"# Note: All other functions are imported directly:",
		"# sphere, rosenbrock, ackley, rastrigin, schwefel, griewank,",
		"# michalewicz, beale, booth, matyas, branin, himmelblau,",
		"# goldstein_price, easom",
	}, "\n")
}

func (bo *BO) customFunctionCode() string {
	if bo.Objective != "custom" || bo.CustomFunction == nil {
		return ""
	}
	return fmt.Sprintf("\n# Custom Function\n%s\n", bo.CustomFunction.Code)
}

func (bo *BO) buildKernel() string {
	if bo.Surrogate != "gp" || bo.Kernel == nil {
		return ""
	}

	kernelMap := map[string]string{
		"rbf":                "RBF(length_scale=1.0)",
		"matern_2.5":         "Matern(nu=2.5)",
		"matern_1.5":         "Matern(nu=1.5)",
		"rational_quadratic": "RationalQuadratic()",
		"exp_sine_squared":   "ExpSineSquared()",
		"dot_product":        "DotProduct()",
		"white":              "WhiteKernel()",
		"constant":           "ConstantKernel()",
	}

	return fmt.Sprintf("\nkernel = %s\n", kernelMap[*bo.Kernel])
}

func (bo *BO) loggingCode() string {
	if !bo.Params.Verbose {
		return ""
	}

	return strings.Join([]string{
		"\n# Logging infrastructure",
		"logbook = []",
		"",
		"def wrapped_with_logging(func, direction):",
		"    \"\"\"Wrapper that logs each evaluation\"\"\"",
		"    def wrapper(x):",
		"        val = float(func(np.asarray(x)))",
		"        result = val if direction == 'minimize' else -val",
		"        ",
		"        # Log this evaluation",
		"        logbook.append({",
		"            'point': list(x),",
		"            'f(x)': val,",
		"        })",
		"        ",
		"        return result",
		"    ",
		"    return wrapper",
	}, "\n")
}

func (bo *BO) mainFunction() string {
	var code strings.Builder

	code.WriteString("\ndef main():")
	if bo.Params.Verbose {
		code.WriteString("\n    global logbook")
		code.WriteString("\n    logbook = []")
	}
	code.WriteString("\n    rootPath = os.path.dirname(os.path.abspath(__file__))\n")

	// Dimensions and bounds
	dims := len(bo.Bounds)
	code.WriteString(fmt.Sprintf("    dimensions = %d\n", dims))
	code.WriteString("    bounds = [\n")
	for _, bound := range bo.Bounds {
		code.WriteString(fmt.Sprintf("        (%f, %f),\n", bound.Min, bound.Max))
	}
	code.WriteString("    ]\n\n")

	// Select objective function
	code.WriteString("    # Objective function\n")
	if bo.Objective == "custom" {
		code.WriteString(fmt.Sprintf("    func = %s\n", bo.CustomFunction.Name))
	} else {
		code.WriteString(fmt.Sprintf("    func = %s\n", bo.Objective))
	}

	// Wrapped objective
	code.WriteString(fmt.Sprintf("    direction = '%s'\n", bo.Direction))
	if bo.Params.Verbose {
		code.WriteString("    wrapped = wrapped_with_logging(func, direction)\n\n")
	} else {
		code.WriteString("    def wrapped(x):\n")
		code.WriteString("        val = float(func(np.asarray(x)))\n")
		code.WriteString("        return val if direction == 'minimize' else -val\n\n")
	}

	// Build skopt dimensions
	code.WriteString("    sk_dims = [Real(lo, hi, name=f'x{i}') for i, (lo, hi) in enumerate(bounds)]\n")
	code.WriteString("    space = Space(sk_dims)\n\n")

	// Initial design
	code.WriteString(fmt.Sprintf("    # Initial design: %s\n", bo.InitialDesign.Strategy))
	code.WriteString(fmt.Sprintf("    n_initial = %d\n", bo.Params.InitialPoints))

	seed := 42
	if bo.Params.RandomSeed != nil {
		seed = *bo.Params.RandomSeed
	}

	switch bo.InitialDesign.Strategy {
	case "random":
		code.WriteString(fmt.Sprintf("    X0 = space.rvs(n_samples=n_initial, random_state=%d)\n", seed))
	case "lhs":
		lhsType := "centered"
		if bo.InitialDesign.LHSType != nil {
			lhsType = *bo.InitialDesign.LHSType
		}
		criterion := "None"
		if bo.InitialDesign.Criterion != nil {
			criterion = fmt.Sprintf("'%s'", *bo.InitialDesign.Criterion)
		}
		code.WriteString(fmt.Sprintf("    sampler = Lhs(lhs_type='%s', criterion=%s)\n", lhsType, criterion))
		code.WriteString("    X0 = sampler.generate(space, n_initial)\n")
	case "sobol":
		code.WriteString("    sampler = Sobol()\n")
		code.WriteString("    X0 = sampler.generate(space, n_initial)\n")
	case "halton":
		code.WriteString("    sampler = Halton()\n")
		code.WriteString("    X0 = sampler.generate(space, n_initial)\n")
	case "hammersley":
		code.WriteString("    sampler = Hammersly()\n")
		code.WriteString("    X0 = sampler.generate(space, n_initial)\n")
	case "grid":
		code.WriteString("    sampler = Grid()\n")
		code.WriteString("    X0 = sampler.generate(space, n_initial)\n")
	}
	code.WriteString("    x0_list = [list(row) for row in X0]\n\n")

	// Acquisition function
	acqMap := map[string]string{"ei": "EI", "pi": "PI", "lcb": "LCB"}
	code.WriteString(fmt.Sprintf("    acq_func = '%s'\n", acqMap[bo.Acquisition]))

	// Parameters
	if bo.Acquisition == "ei" || bo.Acquisition == "pi" {
		xi := 0.01
		if bo.Params.Xi != nil {
			xi = *bo.Params.Xi
		}
		code.WriteString(fmt.Sprintf("    xi = %f\n", xi))
	}
	if bo.Acquisition == "lcb" {
		kappa := 2.576
		if bo.Params.Kappa != nil {
			kappa = *bo.Params.Kappa
		}
		code.WriteString(fmt.Sprintf("    kappa = %f\n", kappa))
	}

	// Base estimator and optimizer
	code.WriteString("\n    # Surrogate model\n")
	if bo.Surrogate == "gp" {
		code.WriteString("    base_estimator = GaussianProcessRegressor(kernel=kernel, normalize_y=True, n_restarts_optimizer=5)\n")
		code.WriteString("    optimizer_callable = gp_minimize\n")
	} else {
		code.WriteString("    base_estimator = 'RF'\n")
		code.WriteString("    optimizer_callable = forest_minimize\n")
	}

	// Print configuration if verbose
	if bo.Params.Verbose {
		code.WriteString("\n    # Print configuration\n")
		code.WriteString("    print('\\n' + '='*80)\n")
		code.WriteString("    print('CONFIGURATION')\n")
		code.WriteString("    print('='*80)\n")
		code.WriteString("    print(f'Function: {func.__name__}')\n")
		code.WriteString("    print(f'Direction: {direction}')\n")
		code.WriteString("    print(f'Dimensions: {dimensions}')\n")
		code.WriteString("    print(f'Bounds: {bounds}')\n")
		code.WriteString(fmt.Sprintf("    print(f'Initial samples (%s): {n_initial}')\n", strings.ToUpper(bo.InitialDesign.Strategy)))
		code.WriteString(fmt.Sprintf("    print(f'Total calls: %d')\n", bo.Params.Iterations))
		code.WriteString("    print(f'Acquisition function: {acq_func}")
		if bo.Acquisition == "ei" || bo.Acquisition == "pi" {
			code.WriteString(" (xi={xi})")
		}
		if bo.Acquisition == "lcb" {
			code.WriteString(" (kappa={kappa})")
		}
		code.WriteString("')\n")
		if bo.Surrogate == "gp" {
			code.WriteString("    print(f'Kernel: {kernel}')\n")
		} else {
			code.WriteString("    print(f'Surrogate: Random Forest')\n")
		}
		code.WriteString("    print('='*80 + '\\n')\n")
	}

	// Evaluate initial points
	code.WriteString("\n    # Evaluate initial points\n")
	code.WriteString("    y0 = [wrapped(x) for x in x0_list]\n\n")

	// Run optimization
	code.WriteString("    # Run Bayesian Optimization\n")
	code.WriteString("    start_time = time.time()\n")
	code.WriteString(fmt.Sprintf("    n_calls = %d\n", bo.Params.Iterations))
	code.WriteString("    res = optimizer_callable(\n")
	code.WriteString("        func=wrapped,\n")
	code.WriteString("        dimensions=sk_dims,\n")
	code.WriteString("        n_calls=n_calls,\n")
	code.WriteString("        n_initial_points=0,\n")
	code.WriteString("        x0=x0_list,\n")
	code.WriteString("        y0=y0,\n")
	code.WriteString("        base_estimator=base_estimator,\n")
	code.WriteString("        acq_func=acq_func,\n")
	if bo.Acquisition == "ei" || bo.Acquisition == "pi" {
		code.WriteString("        xi=xi,\n")
	}
	if bo.Acquisition == "lcb" {
		code.WriteString("        kappa=kappa,\n")
	}

	code.WriteString(fmt.Sprintf("        random_state=%d,\n", seed))
	code.WriteString("        verbose=False\n")
	code.WriteString("    )\n")
	code.WriteString("    elapsed = time.time() - start_time\n\n")

	// Extract results
	code.WriteString("    # Extract best result\n")
	code.WriteString("    all_func_vals = np.array(res.func_vals)\n")
	code.WriteString("    best_idx = np.argmin(all_func_vals)\n")
	code.WriteString("    best_x = res.x_iters[best_idx]\n")
	code.WriteString("    best_fun_skopt = all_func_vals[best_idx]\n")
	code.WriteString("    best_user = best_fun_skopt if direction == 'minimize' else -best_fun_skopt\n\n")

	// Verbose logging output
	if bo.Params.Verbose {
		code.WriteString("    # Compute statistics\n")
		code.WriteString("    fvals = [entry['f(x)'] for entry in logbook]\n")
		code.WriteString("    if direction == 'minimize':\n")
		code.WriteString("        best_entry = min(logbook, key=lambda e: e['f(x)'])\n")
		code.WriteString("    else:\n")
		code.WriteString("        best_entry = max(logbook, key=lambda e: e['f(x)'])\n")
		code.WriteString("    best_idx = logbook.index(best_entry)\n\n")

		code.WriteString("    # Print optimization summary\n")
		code.WriteString("    print('\\n' + '='*80)\n")
		code.WriteString("    print('OPTIMIZATION SUMMARY')\n")
		code.WriteString("    print('='*80)\n")
		code.WriteString("    print(f'Total Evaluations: {len(logbook)}')\n")
		code.WriteString("    print(f'Mean f(x): {np.mean(fvals):.6f}')\n")
		code.WriteString("    print(f'Std f(x): {np.std(fvals):.6f}')\n")
		code.WriteString("    print(f'Min f(x): {np.min(fvals):.6f}')\n")
		code.WriteString("    print(f'Max f(x): {np.max(fvals):.6f}')\n")
		code.WriteString("    print(f'Time elapsed: {elapsed:.2f}s')\n")
		code.WriteString("    print('-'*80)\n")
		code.WriteString("    print(f\"Best Result ({'Minimum' if direction == 'minimize' else 'Maximum'}):\")\n")
		code.WriteString("    print(f'  Iteration: {best_idx + 1}')\n")
		code.WriteString("    print(f'  Point: {best_entry[\"point\"]}')\n")
		code.WriteString("    print(f'  Value: {best_entry[\"f(x)\"]:.10f}')\n")
		code.WriteString("    print('='*80 + '\\n')\n\n")

		// Save detailed logbook
		code.WriteString("    # Save detailed logbook\n")
		code.WriteString("    with open(f'{rootPath}/logbook.txt', 'w') as f:\n")
		code.WriteString("        f.write('='*80 + '\\n')\n")
		code.WriteString("        f.write('CONFIGURATION\\n')\n")
		code.WriteString("        f.write('='*80 + '\\n')\n")
		code.WriteString("        f.write(f'Function: {func.__name__}\\n')\n")
		code.WriteString("        f.write(f'Direction: {direction}\\n')\n")
		code.WriteString("        f.write(f'Dimensions: {dimensions}\\n')\n")
		code.WriteString("        f.write(f'Bounds: {bounds}\\n')\n")
		code.WriteString(fmt.Sprintf("        f.write(f'Initial samples (%s): {n_initial}\\n')\n", strings.ToUpper(bo.InitialDesign.Strategy)))
		code.WriteString("        f.write(f'Total calls: {n_calls}\\n')\n")
		code.WriteString("        f.write(f'Acquisition function: {acq_func}")
		if bo.Acquisition == "ei" || bo.Acquisition == "pi" {
			code.WriteString(" (xi={xi})")
		}
		if bo.Acquisition == "lcb" {
			code.WriteString(" (kappa={kappa})")
		}
		code.WriteString("\\n')\n")
		if bo.Surrogate == "gp" {
			code.WriteString("        f.write(f'Kernel: {kernel}\\n')\n")
		} else {
			code.WriteString("        f.write(f'Surrogate: Random Forest\\n')\n")
		}
		code.WriteString("        f.write('='*80 + '\\n\\n')\n")
		code.WriteString("        \n")
		code.WriteString("        f.write('='*80 + '\\n')\n")
		code.WriteString("        f.write('ITERATION LOG\\n')\n")
		code.WriteString("        f.write('='*80 + '\\n')\n")
		code.WriteString("        f.write(f\"{'Iter':<6} {'Point':<45} {'f(x)':<15} {'Type':<10}\\n\")\n")
		code.WriteString("        f.write('-'*80 + '\\n')\n")
		code.WriteString("        \n")
		code.WriteString("        for i, entry in enumerate(logbook, 1):\n")
		code.WriteString("            point_str = '[' + ', '.join([f'{v:8.4f}' for v in entry['point']]) + ']'\n")
		code.WriteString("            iter_type = 'Initial' if i <= n_initial else 'BO'\n")
		code.WriteString("            f.write(f\"{i:<6} {point_str:<45} {entry['f(x)']:<15.6f} {iter_type:<10}\\n\")\n")
		code.WriteString("        \n")
		code.WriteString("        f.write('='*80 + '\\n\\n')\n")
		code.WriteString("        \n")
		code.WriteString("        f.write('='*80 + '\\n')\n")
		code.WriteString("        f.write('OPTIMIZATION SUMMARY\\n')\n")
		code.WriteString("        f.write('='*80 + '\\n')\n")
		code.WriteString("        f.write(f'Total Evaluations: {len(logbook)}\\n')\n")
		code.WriteString("        f.write(f'Mean f(x): {np.mean(fvals):.6f}\\n')\n")
		code.WriteString("        f.write(f'Std f(x): {np.std(fvals):.6f}\\n')\n")
		code.WriteString("        f.write(f'Min f(x): {np.min(fvals):.6f}\\n')\n")
		code.WriteString("        f.write(f'Max f(x): {np.max(fvals):.6f}\\n')\n")
		code.WriteString("        f.write(f'Time elapsed: {elapsed:.2f}s\\n')\n")
		code.WriteString("        f.write('-'*80 + '\\n')\n")
		code.WriteString("        f.write(f\"Best Result ({'Minimum' if direction == 'minimize' else 'Maximum'}):\\n\")\n")
		code.WriteString("        f.write(f'  Iteration: {best_idx + 1}\\n')\n")
		code.WriteString("        f.write(f'  Point: {best_entry[\"point\"]}\\n')\n")
		code.WriteString("        f.write(f'  Value: {best_entry[\"f(x)\"]:.10f}\\n')\n")
		code.WriteString("        f.write('='*80 + '\\n')\n\n")
	}

	// Save convergence animation
	code.WriteString("    # Create convergence animation\n")
	code.WriteString("    ys = np.array(res.func_vals)\n")
	code.WriteString("    if direction == 'maximize':\n")
	code.WriteString("        ys = -ys\n")
	code.WriteString("    best_vals = np.minimum.accumulate(ys) if direction == 'minimize' else np.maximum.accumulate(ys)\n")
	code.WriteString("    \n")
	code.WriteString("    fig, ax = plt.subplots(figsize=(8, 5))\n")
	code.WriteString("    line, = ax.plot([], [], 'o-', linewidth=2, markersize=4, color='#2E86AB')\n")
	code.WriteString("    \n")
	code.WriteString("    # Dynamic axis limits with proper padding\n")
	code.WriteString("    y_range = max(best_vals) - min(best_vals)\n")
	code.WriteString("    y_padding = max(y_range * 0.1, abs(min(best_vals)) * 0.05) if y_range > 0 else 1\n")
	code.WriteString("    ax.set_xlim(0, len(best_vals) + 1)\n")
	code.WriteString("    ax.set_ylim(min(best_vals) - y_padding, max(best_vals) + y_padding)\n")
	code.WriteString("    \n")
	code.WriteString("    ax.set_xlabel('Number of Calls', fontsize=12, fontweight='bold')\n")
	code.WriteString("    ax.set_ylabel('Best f(x) Found', fontsize=12, fontweight='bold')\n")
	code.WriteString("    ax.set_title('Bayesian Optimization Convergence', fontsize=14, fontweight='bold', pad=20)\n")
	code.WriteString("    ax.grid(True, alpha=0.3, linestyle='--')\n")
	code.WriteString("    \n")
	code.WriteString("    def update_frame(frame):\n")
	code.WriteString("        line.set_data(range(1, frame + 2), best_vals[:frame + 1])\n")
	code.WriteString("        return line,\n")
	code.WriteString("    \n")
	code.WriteString("    ani = animation.FuncAnimation(fig, update_frame, frames=len(best_vals), blit=True, repeat=False)\n")
	code.WriteString("    ani.save(f'{rootPath}/convergence.gif', writer='pillow', fps=10)\n")
	code.WriteString("    plt.close()\n\n")

	// Save results
	code.WriteString("    # Save results\n")
	code.WriteString("    with open(f'{rootPath}/best.txt', 'w') as outfile:\n")
	code.WriteString("        outfile.write(f'Best point: {best_x}\\n')\n")
	code.WriteString("        outfile.write(f'Best value: {best_user}\\n')\n")
	code.WriteString("        outfile.write(f'Total evaluations: {len(res.x_iters)}\\n')\n")
	code.WriteString("        outfile.write(f'Time elapsed: {elapsed:.2f}s\\n')\n")

	if bo.Params.Verbose {
		code.WriteString("\n    print(f'\\nResults saved to:')\n")
		code.WriteString("    print(f'  - {rootPath}/logbook.txt')\n")
		code.WriteString("    print(f'  - {rootPath}/best.txt')\n")
		code.WriteString("    print(f'  - {rootPath}/convergence.gif\\n')\n")
	}

	code.WriteString("\n")
	return code.String()
}

func (bo *BO) Code() (string, error) {
	if err := bo.validate(); err != nil {
		return "", err
	}

	var code strings.Builder

	code.WriteString(bo.imports())
	code.WriteString(bo.benchmarkFunctions())
	code.WriteString(bo.customFunctionCode())
	code.WriteString(bo.buildKernel())
	code.WriteString(bo.loggingCode())
	code.WriteString(bo.mainFunction())
	code.WriteString("\nif __name__ == '__main__':\n")
	code.WriteString("    main()\n")

	return code.String(), nil
}
