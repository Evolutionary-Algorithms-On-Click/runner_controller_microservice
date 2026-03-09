package modules

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
)

// ==================== MOBO Types ====================

type MOBOBound struct {
	Min float64 `json:"min"`
	Max float64 `json:"max"`
}

type MOBOInitialDesign struct {
	Strategy     string  `json:"strategy"`
	Scramble     bool    `json:"scramble"`
	Optimization *string `json:"optimization"`
	Power2       *bool   `json:"power2"`   // Sobol only
	Strength     *int    `json:"strength"` // LHS only
}

type MOBOMCSampler struct {
	Type    string `json:"type"`    // "sobol_qmc" or "iid"
	Samples int    `json:"samples"` // number of MC samples
}

type MOBORefPoint struct {
	UseHeuristic bool       `json:"useHeuristic"`
	Values       *[]float64 `json:"values"`
}

type MOBOModelConfig struct {
	Architecture string   `json:"architecture"` // "independent" or "joint"
	ModelType    string   `json:"modelType"`    // "single_task", "single_task_fixed_noise", "multi_task"
	NoiseLevel   *float64 `json:"noiseLevel"`   // for fixed noise mode
}

type MOBOParams struct {
	InitialPoints int      `json:"initialPoints"`
	Iterations    int      `json:"iterations"`
	BatchSize     int      `json:"batchSize"`
	Restarts      int      `json:"restarts"`
	RawSamples    int      `json:"rawSamples"`
	Beta          *float64 `json:"beta"` // for qUCB variants
	RandomSeed    *int     `json:"randomSeed"`
	Verbose       bool     `json:"verbose"`
}

type MOBOCustomFunction struct {
	Code          string `json:"code"`
	Name          string `json:"name"`
	Dim           int    `json:"dim"`
	NumObjectives int    `json:"numObjectives"`
	Description   string `json:"description"`
}

type MOBOProblemConfig struct {
	Dim        interface{} `json:"dim"`        // can be int or "configurable"
	Objectives interface{} `json:"objectives"` // can be int or "configurable"
}

type MOBO struct {
	AlgorithmType  string              `json:"algorithm_type"`
	Problem        string              `json:"problem"`
	ProblemConfig  *MOBOProblemConfig  `json:"problem_config"`
	CustomFunction *MOBOCustomFunction `json:"custom_function"`
	ModelConfig    MOBOModelConfig     `json:"model_config"`
	Acquisition    string              `json:"acquisition"`
	Bounds         []MOBOBound         `json:"bounds"`
	InitialDesign  MOBOInitialDesign   `json:"initial_design"`
	MCSampler      MOBOMCSampler       `json:"mc_sampler"`
	RefPoint       MOBORefPoint        `json:"ref_point"`
	Params         MOBOParams          `json:"params"`
}

func MOBOFromJSON(jsonData map[string]any) (*MOBO, error) {
	mobo := &MOBO{}
	jsonDataBytes, err := json.Marshal(jsonData)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(jsonDataBytes, mobo); err != nil {
		return nil, err
	}
	return mobo, nil
}

func pythonBool(b bool) string {
	if b {
		return "True"
	}
	return "False"
}

func (mobo *MOBO) validate() error {
	// Validate algorithm type
	if mobo.AlgorithmType != "mobo" {
		return fmt.Errorf("invalid algorithm type: %s", mobo.AlgorithmType)
	}

	// Validate problem
	validProblems := []string{
		"branin_currin", "dtlz1", "dtlz2", "dtlz3", "dtlz4", "dtlz5", "dtlz7",
		"zdt1", "zdt2", "zdt3", "vehicle_safety", "car_side_impact", "custom",
	}
	if !slices.Contains(validProblems, mobo.Problem) {
		return fmt.Errorf("invalid problem: %s", mobo.Problem)
	}

	// Validate custom function if problem is custom
	if mobo.Problem == "custom" {
		if mobo.CustomFunction == nil {
			return fmt.Errorf("custom function required when problem is 'custom'")
		}
		if mobo.CustomFunction.Code == "" || mobo.CustomFunction.Name == "" {
			return fmt.Errorf("custom function code and name are required")
		}
		if mobo.CustomFunction.Dim < 1 {
			return fmt.Errorf("custom function dimension must be at least 1")
		}
		if mobo.CustomFunction.NumObjectives < 2 {
			return fmt.Errorf("custom function must have at least 2 objectives")
		}
	}

	// Validate model config
	if !slices.Contains([]string{"independent", "joint"}, mobo.ModelConfig.Architecture) {
		return fmt.Errorf("invalid model architecture: %s", mobo.ModelConfig.Architecture)
	}

	validModelTypes := []string{"single_task", "single_task_fixed_noise", "multi_task"}
	if !slices.Contains(validModelTypes, mobo.ModelConfig.ModelType) {
		return fmt.Errorf("invalid model type: %s", mobo.ModelConfig.ModelType)
	}

	// Joint architecture must use multi_task
	if mobo.ModelConfig.Architecture == "joint" && mobo.ModelConfig.ModelType != "multi_task" {
		return fmt.Errorf("joint architecture requires multi_task model type")
	}

	// Validate noise level for fixed noise mode
	if mobo.ModelConfig.ModelType == "single_task_fixed_noise" {
		if mobo.ModelConfig.NoiseLevel == nil {
			return fmt.Errorf("noise level required for single_task_fixed_noise model")
		}
		if *mobo.ModelConfig.NoiseLevel <= 0 {
			return fmt.Errorf("noise level must be positive")
		}
	}

	// Validate acquisition function
	validAcquisitions := []string{
		"qnehvi", "qehvi", "qlog_nehvi", "qlog_ehvi",
		"qlog_nparego", "qnparego",
		"scalarized_qei", "scalarized_qnei", "scalarized_qucb", "scalarized_qpi", "scalarized_qsr",
		"weighted_sum_qei",
	}
	if !slices.Contains(validAcquisitions, mobo.Acquisition) {
		return fmt.Errorf("invalid acquisition function: %s", mobo.Acquisition)
	}

	// Validate bounds
	if len(mobo.Bounds) == 0 {
		return fmt.Errorf("at least one dimension bound is required")
	}
	for i, bound := range mobo.Bounds {
		if bound.Min >= bound.Max {
			return fmt.Errorf("invalid bounds for dimension %d: min=%f, max=%f", i, bound.Min, bound.Max)
		}
	}

	// Validate initial design
	validStrategies := []string{"sobol", "lhs", "halton", "random"}
	if !slices.Contains(validStrategies, mobo.InitialDesign.Strategy) {
		return fmt.Errorf("invalid initial design strategy: %s", mobo.InitialDesign.Strategy)
	}

	// Validate MC sampler
	if !slices.Contains([]string{"sobol_qmc", "iid"}, mobo.MCSampler.Type) {
		return fmt.Errorf("invalid MC sampler type: %s", mobo.MCSampler.Type)
	}
	if mobo.MCSampler.Samples < 32 {
		return fmt.Errorf("MC samples must be at least 32")
	}

	// Validate reference point
	if !mobo.RefPoint.UseHeuristic {
		if mobo.RefPoint.Values == nil {
			return fmt.Errorf("reference point values required when not using heuristic")
		}
	}

	// Validate params
	if mobo.Params.InitialPoints < 2 {
		return fmt.Errorf("initial points must be at least 2")
	}
	if mobo.Params.Iterations < 1 {
		return fmt.Errorf("iterations must be at least 1")
	}
	if mobo.Params.BatchSize < 1 {
		return fmt.Errorf("batch size must be at least 1")
	}
	if mobo.Params.Restarts < 1 {
		return fmt.Errorf("restarts must be at least 1")
	}
	if mobo.Params.RawSamples < 64 {
		return fmt.Errorf("raw samples must be at least 64")
	}

	return nil
}

func (mobo *MOBO) imports() string {
	return strings.Join([]string{
		"#!/usr/bin/env python3",
		"import os",
		"import uuid",
		"import math",
		"import time",
		"import warnings",
		"from typing import Dict, List, Optional, Tuple, Callable",
		"",
		"import numpy as np",
		"import torch",
		"from matplotlib import pyplot as plt",
		"import matplotlib",
		"matplotlib.use('Agg')",
		"from scipy.stats import qmc",
		"",
		"# BoTorch / GPyTorch imports",
		"from botorch.acquisition import (",
		"    qExpectedImprovement,",
		"    qNoisyExpectedImprovement,",
		"    qProbabilityOfImprovement,",
		"    qSimpleRegret,",
		"    qUpperConfidenceBound,",
		")",
		"from botorch.acquisition.multi_objective import (",
		"    qExpectedHypervolumeImprovement,",
		"    qLogExpectedHypervolumeImprovement,",
		"    qLogNoisyExpectedHypervolumeImprovement,",
		"    qNoisyExpectedHypervolumeImprovement,",
		")",
		"from botorch.acquisition.multi_objective.parego import qLogNParEGO",
		"from botorch.acquisition.objective import (",
		"    GenericMCObjective,",
		"    LinearMCObjective,",
		"    ScalarizedPosteriorTransform,",
		")",
		"from botorch.fit import fit_gpytorch_mll",
		"from botorch.models import ModelListGP, SingleTaskGP, MultiTaskGP",
		"from botorch.optim import optimize_acqf",
		"from botorch.sampling import IIDNormalSampler, SobolQMCNormalSampler",
		"from botorch.utils.multi_objective.box_decompositions.dominated import (",
		"    DominatedPartitioning,",
		")",
		"from botorch.utils.multi_objective.pareto import is_non_dominated",
		"from botorch.utils.multi_objective.scalarization import get_chebyshev_scalarization",
		"from botorch.utils.sampling import sample_simplex",
		"",
		"from gpytorch.mlls import ExactMarginalLogLikelihood",
		"",
		"# Multi-objective test functions",
		"from botorch.test_functions.multi_objective import (",
		"    BraninCurrin,",
		"    CarSideImpact,",
		"    DTLZ1,",
		"    DTLZ2,",
		"    DTLZ3,",
		"    DTLZ4,",
		"    DTLZ5,",
		"    DTLZ7,",
		"    VehicleSafety,",
		"    ZDT1,",
		"    ZDT2,",
		"    ZDT3,",
		")",
		"",
		"from PIL import Image",
		"",
		"warnings.filterwarnings('ignore')",
		"torch.set_default_dtype(torch.double)",
	}, "\n")
}

func (mobo *MOBO) customFunctionCode() string {
	if mobo.Problem != "custom" || mobo.CustomFunction == nil {
		return ""
	}

	var code strings.Builder
	code.WriteString("\n# Custom Multi-Objective Function\n")
	code.WriteString("class CustomMultiObjectiveFunction:\n")
	code.WriteString("    \"\"\"Wrapper for custom user-defined multi-objective functions.\"\"\"\n\n")
	code.WriteString("    def __init__(self, code: str, func_name: str, dim: int, num_objectives: int, bounds: torch.Tensor):\n")
	code.WriteString("        self.code = code\n")
	code.WriteString("        self.func_name = func_name\n")
	code.WriteString("        self.dim = dim\n")
	code.WriteString("        self.num_objectives = num_objectives\n")
	code.WriteString("        self.bounds = bounds\n")
	code.WriteString("        self._func = self._load_function()\n\n")
	code.WriteString("    def _load_function(self) -> Callable:\n")
	code.WriteString("        namespace = {'np': np, 'numpy': np, 'math': math, 'torch': torch}\n")
	code.WriteString("        exec(self.code, namespace)\n")
	code.WriteString("        if self.func_name not in namespace:\n")
	code.WriteString("            raise ValueError(f\"Function '{self.func_name}' not found\")\n")
	code.WriteString("        func = namespace[self.func_name]\n")
	code.WriteString("        if not callable(func):\n")
	code.WriteString("            raise ValueError(f\"'{self.func_name}' is not callable\")\n")
	code.WriteString("        return func\n\n")
	code.WriteString("    def __call__(self, X: torch.Tensor) -> torch.Tensor:\n")
	code.WriteString("        original_device = X.device\n")
	code.WriteString("        original_dtype = X.dtype\n")
	code.WriteString("        if X.dim() == 1:\n")
	code.WriteString("            X = X.unsqueeze(0)\n")
	code.WriteString("        results = []\n")
	code.WriteString("        for x in X:\n")
	code.WriteString("            x_np = x.detach().cpu().numpy()\n")
	code.WriteString("            result = self._func(x_np)\n")
	code.WriteString("            result = np.atleast_1d(result)\n")
	code.WriteString("            if len(result) != self.num_objectives:\n")
	code.WriteString("                raise ValueError(f\"Expected {self.num_objectives} objectives, got {len(result)}\")\n")
	code.WriteString("            results.append(result)\n")
	code.WriteString("        results_array = np.array(results)\n")
	code.WriteString("        return torch.tensor(results_array, dtype=original_dtype, device=original_device)\n\n")

	// Add the actual custom function code
	code.WriteString(mobo.CustomFunction.Code)
	code.WriteString("\n\n")

	return code.String()
}

func (mobo *MOBO) helperFunctions() string {
	return strings.Join([]string{
		"\n# Helper Functions",
		"def compute_hypervolume(Y: torch.Tensor, ref_point: torch.Tensor) -> float:",
		"    \"\"\"Compute hypervolume of non-dominated set.\"\"\"",
		"    pareto_mask = is_non_dominated(Y)",
		"    pareto_Y = Y[pareto_mask]",
		"    if pareto_Y.numel() == 0:",
		"        return 0.0",
		"    bd = DominatedPartitioning(ref_point=ref_point, Y=pareto_Y)",
		"    return bd.compute_hypervolume().item()",
		"",
		"def get_scalarization_weights(num_objectives: int, method: str = 'random', seed: Optional[int] = None) -> torch.Tensor:",
		"    \"\"\"Generate scalarization weights.\"\"\"",
		"    if method == 'equal':",
		"        return torch.ones(num_objectives, dtype=torch.double) / num_objectives",
		"    elif method == 'random':",
		"        if seed is not None:",
		"            torch.manual_seed(seed)",
		"        return sample_simplex(num_objectives, n=1).squeeze(0)",
		"    else:",
		"        raise ValueError(f'Unknown weight method {method}')",
		"",
		"def plot_hypervolume(hv_history: List[float], save_path: str):",
		"    \"\"\"Plot hypervolume convergence.\"\"\"",
		"    fig, ax = plt.subplots(figsize=(8, 5))",
		"    iters = np.arange(len(hv_history))",
		"    ax.plot(iters, hv_history, 'o-', linewidth=2, markersize=4, color='#2E86AB')",
		"    ax.set_xlabel('Iteration', fontsize=12, fontweight='bold')",
		"    ax.set_ylabel('Hypervolume', fontsize=12, fontweight='bold')",
		"    ax.set_title('Hypervolume Convergence', fontsize=14, fontweight='bold', pad=20)",
		"    ax.grid(True, alpha=0.3, linestyle='--')",
		"    plt.tight_layout()",
		"    plt.savefig(save_path, dpi=150)",
		"    plt.close(fig)",
		"",
		"def plot_pareto_front_2d(Y_all: torch.Tensor, ref_point: torch.Tensor, save_path: str, iteration: int):",
		"    \"\"\"Plot 2D Pareto front.\"\"\"",
		"    fig, ax = plt.subplots(figsize=(7, 5))",
		"    Y_np = Y_all.cpu().numpy()",
		"    ax.scatter(Y_np[:, 0], Y_np[:, 1], alpha=0.4, s=25, label='All points')",
		"    pareto_mask = is_non_dominated(Y_all)",
		"    pareto_Y = Y_all[pareto_mask].cpu().numpy()",
		"    if len(pareto_Y) > 0:",
		"        idx = np.argsort(pareto_Y[:, 0])",
		"        pf = pareto_Y[idx]",
		"        ax.plot(pf[:, 0], pf[:, 1], '-', linewidth=2, label='Pareto front')",
		"        ax.scatter(pf[:, 0], pf[:, 1], s=80, marker='*', edgecolors='black', linewidths=1.0, label='Pareto points')",
		"    ref = ref_point.cpu().numpy()",
		"    ax.scatter(ref[0], ref[1], marker='x', s=120, linewidths=2, label='Ref point')",
		"    ax.set_xlabel('Objective 1')",
		"    ax.set_ylabel('Objective 2')",
		"    ax.set_title(f'Pareto Front (Iteration {iteration})')",
		"    ax.grid(True, alpha=0.3, linestyle='--')",
		"    ax.legend()",
		"    plt.tight_layout()",
		"    plt.savefig(save_path, dpi=150)",
		"    plt.close(fig)",
		"",
		"def plot_pareto_front_3d(Y_all: torch.Tensor, ref_point: torch.Tensor, save_path: str, iteration: int):",
		"    \"\"\"Plot 3D Pareto front.\"\"\"",
		"    fig = plt.figure(figsize=(8, 6))",
		"    ax = fig.add_subplot(111, projection='3d')",
		"    Y_np = Y_all.cpu().numpy()",
		"    ax.scatter(Y_np[:, 0], Y_np[:, 1], Y_np[:, 2], alpha=0.35, s=25, label='All points')",
		"    pareto_mask = is_non_dominated(Y_all)",
		"    pareto_Y = Y_all[pareto_mask].cpu().numpy()",
		"    if len(pareto_Y) > 0:",
		"        ax.scatter(pareto_Y[:, 0], pareto_Y[:, 1], pareto_Y[:, 2], s=80, marker='*', edgecolors='black', linewidths=1.0, label='Pareto points')",
		"    ref = ref_point.cpu().numpy()",
		"    ax.scatter(ref[0], ref[1], ref[2], marker='x', s=120, linewidths=2, label='Ref point')",
		"    ax.set_xlabel('Objective 1')",
		"    ax.set_ylabel('Objective 2')",
		"    ax.set_zlabel('Objective 3')",
		"    ax.set_title(f'Pareto Front (Iteration {iteration})')",
		"    ax.legend()",
		"    plt.tight_layout()",
		"    plt.savefig(save_path, dpi=150)",
		"    plt.close(fig)",
	}, "\n")
}

func (mobo *MOBO) generateInitialData() string {
	var code strings.Builder

	code.WriteString("\ndef generate_initial_data(problem, n_samples, init_strategy, init_params, seed):\n")
	code.WriteString("    \"\"\"Generate initial training data using chosen sampling strategy.\"\"\"\n")
	code.WriteString("    if seed is not None:\n")
	code.WriteString("        np.random.seed(seed)\n")
	code.WriteString("        torch.manual_seed(seed)\n\n")

	code.WriteString("    dim = problem.dim\n")
	code.WriteString("    bounds = problem.bounds.numpy()\n\n")

	code.WriteString("    # QMC sampling based on strategy\n")
	code.WriteString("    if init_strategy == 'sobol':\n")
	code.WriteString("        sampler_kwargs = {'d': dim, 'scramble': init_params.get('scramble', True), 'optimization': init_params.get('optimization', None)}\n")
	code.WriteString("        if seed is not None:\n")
	code.WriteString("            sampler_kwargs['seed'] = seed\n")
	code.WriteString("        sampler = qmc.Sobol(**sampler_kwargs)\n")
	code.WriteString("        if init_params.get('power2', True):\n")
	code.WriteString("            m = int(np.log2(n_samples))\n")
	code.WriteString("            sample = sampler.random_base2(m=m)\n")
	code.WriteString("        else:\n")
	code.WriteString("            sample = sampler.random(n=n_samples)\n\n")

	code.WriteString("    elif init_strategy == 'lhs':\n")
	code.WriteString("        sampler_kwargs = {'d': dim, 'scramble': init_params.get('scramble', True), 'strength': init_params.get('strength', 1), 'optimization': init_params.get('optimization', None)}\n")
	code.WriteString("        if seed is not None:\n")
	code.WriteString("            sampler_kwargs['seed'] = seed\n")
	code.WriteString("        sampler = qmc.LatinHypercube(**sampler_kwargs)\n")
	code.WriteString("        sample = sampler.random(n=n_samples)\n\n")

	code.WriteString("    elif init_strategy == 'halton':\n")
	code.WriteString("        sampler_kwargs = {'d': dim, 'scramble': init_params.get('scramble', True), 'optimization': init_params.get('optimization', None)}\n")
	code.WriteString("        if seed is not None:\n")
	code.WriteString("            sampler_kwargs['seed'] = seed\n")
	code.WriteString("        sampler = qmc.Halton(**sampler_kwargs)\n")
	code.WriteString("        sample = sampler.random(n=n_samples)\n\n")

	code.WriteString("    else:  # random\n")
	code.WriteString("        sample = np.random.rand(n_samples, dim)\n\n")

	code.WriteString("    # Scale to bounds\n")
	code.WriteString("    sample_scaled = qmc.scale(sample, bounds[0], bounds[1])\n")
	code.WriteString("    X = torch.tensor(sample_scaled, dtype=torch.double)\n")
	code.WriteString("    Y = problem(X)\n")
	code.WriteString("    return X, Y\n")

	return code.String()
}

func (mobo *MOBO) initializeModel() string {
	return strings.Join([]string{
		"\ndef initialize_model(X, Y, architecture, model_type, train_Yvar=None):",
		"    \"\"\"Initialize GP model(s) for multi-objective optimization.\"\"\"",
		"    if architecture == 'joint':",
		"        # MultiTaskGP",
		"        X_list = []",
		"        Y_list = []",
		"        for i in range(Y.shape[-1]):",
		"            task_col = torch.full((X.shape[0], 1), i, dtype=torch.long, device=X.device)",
		"            X_list.append(torch.cat([X, task_col], dim=-1))",
		"            Y_list.append(Y[:, i:i+1])",
		"        X_aug = torch.cat(X_list, dim=0)",
		"        Y_aug = torch.cat(Y_list, dim=0)",
		"        model = MultiTaskGP(X_aug, Y_aug, task_feature=-1, train_Yvar=train_Yvar)",
		"        mll = ExactMarginalLogLikelihood(model.likelihood, model)",
		"        fit_gpytorch_mll(mll)",
		"        return model",
		"    else:",
		"        # Independent models (ModelListGP)",
		"        models = []",
		"        for i in range(Y.shape[-1]):",
		"            y_i = Y[..., i:i+1]",
		"            if model_type == 'single_task':",
		"                m = SingleTaskGP(X, y_i)",
		"            elif model_type == 'single_task_fixed_noise':",
		"                if train_Yvar is None:",
		"                    raise ValueError('train_Yvar required for fixed noise mode')",
		"                yvar_i = train_Yvar[..., i:i+1]",
		"                m = SingleTaskGP(X, y_i, train_Yvar=yvar_i)",
		"            else:",
		"                raise ValueError(f'Unknown model_type: {model_type}')",
		"            mll = ExactMarginalLogLikelihood(m.likelihood, m)",
		"            fit_gpytorch_mll(mll)",
		"            models.append(m)",
		"        return ModelListGP(*models)",
	}, "\n")
}

func (mobo *MOBO) acquisitionFunction() string {
	return strings.Join([]string{
		"\ndef get_acquisition_function(",
		"    model,",
		"    X,",
		"    Y,",
		"    ref_point,",
		"    bounds,",
		"    acq_type,",
		"    sampler,",
		"    batch_size,",
		"    num_objectives,",
		"    beta: float = 0.2,",
		"    seed: Optional[int] = None,",
		"):",
		"    \"\"\"Build the chosen acquisition function.",
		"",
		"    All objectives are treated as maximization (we instantiate test functions with negate=True).",
		"    \"\"\"",
		"    # 1) Hypervolume-based (native MOBO)",
		"    if acq_type == 'qnehvi':",
		"        return qNoisyExpectedHypervolumeImprovement(",
		"            model=model,",
		"            ref_point=ref_point.tolist(),",
		"            X_baseline=X,",
		"            prune_baseline=True,",
		"            sampler=sampler,",
		"        )",
		"",
		"    if acq_type == 'qehvi':",
		"        pareto_mask = is_non_dominated(Y)",
		"        pareto_Y = Y[pareto_mask]",
		"        partitioning = DominatedPartitioning(ref_point=ref_point, Y=pareto_Y)",
		"        return qExpectedHypervolumeImprovement(",
		"            model=model,",
		"            ref_point=ref_point.tolist(),",
		"            partitioning=partitioning,",
		"            sampler=sampler,",
		"        )",
		"",
		"    if acq_type == 'qlog_nehvi':",
		"        return qLogNoisyExpectedHypervolumeImprovement(",
		"            model=model,",
		"            ref_point=ref_point.tolist(),",
		"            X_baseline=X,",
		"            prune_baseline=True,",
		"            sampler=sampler,",
		"        )",
		"",
		"    if acq_type == 'qlog_ehvi':",
		"        pareto_mask = is_non_dominated(Y)",
		"        pareto_Y = Y[pareto_mask]",
		"        partitioning = DominatedPartitioning(ref_point=ref_point, Y=pareto_Y)",
		"        return qLogExpectedHypervolumeImprovement(",
		"            model=model,",
		"            ref_point=ref_point.tolist(),",
		"            partitioning=partitioning,",
		"            sampler=sampler,",
		"        )",
		"",
		"    # 2) ParEGO-style (scalarization + qNEI)",
		"    if acq_type == 'qlog_nparego':",
		"        return qLogNParEGO(",
		"            model=model,",
		"            X_baseline=X,",
		"            sampler=sampler,",
		"            prune_baseline=True,",
		"        )",
		"",
		"    if acq_type == 'qnparego':",
		"        with torch.no_grad():",
		"            pred = model.posterior(X).mean",
		"        weights = get_scalarization_weights(num_objectives, method='random', seed=seed)",
		"        scalarization = get_chebyshev_scalarization(weights=weights, Y=pred)",
		"        objective = GenericMCObjective(scalarization)",
		"        return qNoisyExpectedImprovement(",
		"            model=model,",
		"            objective=objective,",
		"            X_baseline=X,",
		"            sampler=sampler,",
		"            prune_baseline=True,",
		"        )",
		"",
		"    # 3) Scalarized single-objective variants (using posterior transform)",
		"    if acq_type in {",
		"        'scalarized_qei',",
		"        'scalarized_qnei',",
		"        'scalarized_qucb',",
		"        'scalarized_qpi',",
		"        'scalarized_qsr',",
		"    }:",
		"        weights = get_scalarization_weights(num_objectives, method='random', seed=seed)",
		"        posterior_transform = ScalarizedPosteriorTransform(weights=weights)",
		"",
		"        if acq_type == 'scalarized_qei':",
		"            scalarized_y = (Y * weights).sum(dim=-1)",
		"            best_f = scalarized_y.max()",
		"            return qExpectedImprovement(",
		"                model=model,",
		"                best_f=best_f,",
		"                sampler=sampler,",
		"                posterior_transform=posterior_transform,",
		"            )",
		"",
		"        if acq_type == 'scalarized_qnei':",
		"            return qNoisyExpectedImprovement(",
		"                model=model,",
		"                X_baseline=X,",
		"                sampler=sampler,",
		"                posterior_transform=posterior_transform,",
		"                prune_baseline=True,",
		"            )",
		"",
		"        if acq_type == 'scalarized_qucb':",
		"            return qUpperConfidenceBound(",
		"                model=model,",
		"                beta=beta,",
		"                sampler=sampler,",
		"                posterior_transform=posterior_transform,",
		"            )",
		"",
		"        if acq_type == 'scalarized_qpi':",
		"            scalarized_y = (Y * weights).sum(dim=-1)",
		"            best_f = scalarized_y.max()",
		"            return qProbabilityOfImprovement(",
		"                model=model,",
		"                best_f=best_f,",
		"                sampler=sampler,",
		"                posterior_transform=posterior_transform,",
		"            )",
		"",
		"        if acq_type == 'scalarized_qsr':",
		"            return qSimpleRegret(",
		"                model=model,",
		"                sampler=sampler,",
		"                posterior_transform=posterior_transform,",
		"            )",
		"",
		"    # 4) Weighted-sum qEI (LinearMCObjective)",
		"    if acq_type == 'weighted_sum_qei':",
		"        weights = get_scalarization_weights(num_objectives, method='equal', seed=seed)",
		"        objective = LinearMCObjective(weights=weights)",
		"        scalarized_y = (Y * weights).sum(dim=-1)",
		"        best_f = scalarized_y.max()",
		"        return qExpectedImprovement(",
		"            model=model,",
		"            best_f=best_f,",
		"            sampler=sampler,",
		"            objective=objective,",
		"        )",
		"",
		"    raise ValueError(f'Unknown acquisition type: {acq_type}')",
		"",
		"def optimize_acquisition(",
		"    model,",
		"    X,",
		"    Y,",
		"    ref_point,",
		"    bounds,",
		"    acq_type,",
		"    sampler,",
		"    batch_size,",
		"    num_objectives,",
		"    num_restarts: int = 10,",
		"    raw_samples: int = 512,",
		"    beta: float = 0.2,",
		"    seed: Optional[int] = None,",
		"):",
		"    \"\"\"Optimize the selected acquisition function over the given bounds.\"\"\"",
		"    acq_func = get_acquisition_function(",
		"        model=model,",
		"        X=X,",
		"        Y=Y,",
		"        ref_point=ref_point,",
		"        bounds=bounds,",
		"        acq_type=acq_type,",
		"        sampler=sampler,",
		"        batch_size=batch_size,",
		"        num_objectives=num_objectives,",
		"        beta=beta,",
		"        seed=seed,",
		"    )",
		"",
		"    candidates, acq_value = optimize_acqf(",
		"        acq_function=acq_func,",
		"        bounds=bounds,",
		"        q=batch_size,",
		"        num_restarts=num_restarts,",
		"        raw_samples=raw_samples,",
		"        options={'batch_limit': 5, 'maxiter': 200},",
		"        sequential=batch_size > 1,",
		"    )",
		"    return candidates, acq_value",
	}, "\n")
}

func (mobo *MOBO) mainFunction() string {
	var code strings.Builder

	code.WriteString("\ndef main():\n")
	code.WriteString("    rootPath = os.getcwd()\n\n")

	// Problem setup
	code.WriteString("    # Problem setup\n")
	if mobo.Problem == "custom" {
		code.WriteString(fmt.Sprintf("    dim = %d\n", mobo.CustomFunction.Dim))
		code.WriteString(fmt.Sprintf("    M = %d\n", mobo.CustomFunction.NumObjectives))
		code.WriteString("    bounds_list = [\n")
		for _, bound := range mobo.Bounds {
			code.WriteString(fmt.Sprintf("        (%f, %f),\n", bound.Min, bound.Max))
		}
		code.WriteString("    ]\n")
		code.WriteString("    bounds_array = np.array(bounds_list).T\n")
		code.WriteString("    bounds = torch.tensor(bounds_array, dtype=torch.double)\n\n")

		code.WriteString("    # Create custom problem\n")
		code.WriteString(fmt.Sprintf("    custom_code = '''%s'''\n", mobo.CustomFunction.Code))
		code.WriteString("    problem = CustomMultiObjectiveFunction(\n")
		code.WriteString("        code=custom_code,\n")
		code.WriteString(fmt.Sprintf("        func_name='%s',\n", mobo.CustomFunction.Name))
		code.WriteString(fmt.Sprintf("        dim=%d,\n", mobo.CustomFunction.Dim))
		code.WriteString(fmt.Sprintf("        num_objectives=%d,\n", mobo.CustomFunction.NumObjectives))
		code.WriteString("        bounds=bounds\n")
		code.WriteString("    )\n\n")
	} else {
		// Built-in benchmark
		problemMap := map[string]string{
			"branin_currin":   "BraninCurrin",
			"vehicle_safety":  "VehicleSafety",
			"car_side_impact": "CarSideImpact",
			"dtlz1":           "DTLZ1",
			"dtlz2":           "DTLZ2",
			"dtlz3":           "DTLZ3",
			"dtlz4":           "DTLZ4",
			"dtlz5":           "DTLZ5",
			"dtlz7":           "DTLZ7",
			"zdt1":            "ZDT1",
			"zdt2":            "ZDT2",
			"zdt3":            "ZDT3",
		}

		className := problemMap[mobo.Problem]

		// Determine dimensions and objectives
		var dimVal, objVal string
		if mobo.ProblemConfig != nil {
			switch d := mobo.ProblemConfig.Dim.(type) {
			case float64:
				dimVal = fmt.Sprintf("%d", int(d))
			case int:
				dimVal = fmt.Sprintf("%d", d)
			default:
				dimVal = fmt.Sprintf("%d", len(mobo.Bounds))
			}

			switch o := mobo.ProblemConfig.Objectives.(type) {
			case float64:
				objVal = fmt.Sprintf("%d", int(o))
			case int:
				objVal = fmt.Sprintf("%d", o)
			default:
				objVal = "2"
			}
		} else {
			dimVal = fmt.Sprintf("%d", len(mobo.Bounds))
			objVal = "2"
		}

		// Create problem instance
		if strings.HasPrefix(mobo.Problem, "dtlz") {
			code.WriteString(fmt.Sprintf("    problem = %s(dim=%s, num_objectives=%s, negate=True)\n", className, dimVal, objVal))
		} else if strings.HasPrefix(mobo.Problem, "zdt") {
			code.WriteString(fmt.Sprintf("    problem = %s(dim=%s, negate=True)\n", className, dimVal))
		} else {
			code.WriteString(fmt.Sprintf("    problem = %s(negate=True)\n", className))
		}

		code.WriteString("    dim = problem.dim\n")
		code.WriteString("    M = problem.num_objectives\n")
		code.WriteString("    bounds = problem.bounds.to(dtype=torch.double)\n\n")
	}

	// Random seed setup
	seed := 42
	if mobo.Params.RandomSeed != nil {
		seed = *mobo.Params.RandomSeed
	}
	code.WriteString(fmt.Sprintf("    seed = %d\n", seed))
	code.WriteString("    torch.manual_seed(seed)\n")
	code.WriteString("    np.random.seed(seed)\n\n")

	// Configuration summary
	if mobo.Params.Verbose {
		code.WriteString("    # Print configuration\n")
		code.WriteString("    print('\\n' + '='*80)\n")
		code.WriteString("    print('MOBO CONFIGURATION')\n")
		code.WriteString("    print('='*80)\n")
		code.WriteString(fmt.Sprintf("    print('Problem: %s')\n", mobo.Problem))
		code.WriteString("    print(f'Dimensions: {dim}')\n")
		code.WriteString("    print(f'Objectives: {M}')\n")
		code.WriteString(fmt.Sprintf("    print('Model Architecture: %s')\n", mobo.ModelConfig.Architecture))
		code.WriteString(fmt.Sprintf("    print('Model Type: %s')\n", mobo.ModelConfig.ModelType))
		if mobo.ModelConfig.NoiseLevel != nil {
			code.WriteString(fmt.Sprintf("    print('Noise Level: %f')\n", *mobo.ModelConfig.NoiseLevel))
		}
		code.WriteString(fmt.Sprintf("    print('Acquisition: %s')\n", mobo.Acquisition))
		code.WriteString(fmt.Sprintf("    print('Initial Samples: %d (%s)')\n", mobo.Params.InitialPoints, strings.ToUpper(mobo.InitialDesign.Strategy)))
		code.WriteString(fmt.Sprintf("    print('BO Iterations: %d')\n", mobo.Params.Iterations))
		code.WriteString(fmt.Sprintf("    print('Batch Size: %d')\n", mobo.Params.BatchSize))
		code.WriteString(fmt.Sprintf("    print('MC Sampler: %s (%d samples)')\n", mobo.MCSampler.Type, mobo.MCSampler.Samples))
		code.WriteString("    print('='*80 + '\\n')\n\n")
	}

	// Initial design parameters
	code.WriteString("    # Initial design parameters\n")
	code.WriteString("    init_params = {\n")
	code.WriteString(fmt.Sprintf("        'scramble': %v,\n", pythonBool(mobo.InitialDesign.Scramble)))
	if mobo.InitialDesign.Optimization != nil {
		code.WriteString(fmt.Sprintf("        'optimization': '%s',\n", *mobo.InitialDesign.Optimization))
	} else {
		code.WriteString("        'optimization': None,\n")
	}
	if mobo.InitialDesign.Power2 != nil {
		code.WriteString(fmt.Sprintf("        'power2': %v,\n", pythonBool(*mobo.InitialDesign.Power2)))
	}
	if mobo.InitialDesign.Strength != nil {
		code.WriteString(fmt.Sprintf("        'strength': %d,\n", *mobo.InitialDesign.Strength))
	}
	code.WriteString("    }\n\n")

	// Generate initial data
	code.WriteString("    # Generate initial data\n")
	code.WriteString(fmt.Sprintf("    X, Y = generate_initial_data(problem, %d, '%s', init_params, seed)\n\n",
		mobo.Params.InitialPoints, mobo.InitialDesign.Strategy))

	// Setup noise variance if needed
	if mobo.ModelConfig.ModelType == "single_task_fixed_noise" && mobo.ModelConfig.NoiseLevel != nil {
		noiseVar := (*mobo.ModelConfig.NoiseLevel) * (*mobo.ModelConfig.NoiseLevel)
		code.WriteString("    # Fixed noise variance\n")
		code.WriteString(fmt.Sprintf("    train_Yvar = torch.full_like(Y, %f)\n\n", noiseVar))
	} else {
		code.WriteString("    train_Yvar = None\n\n")
	}

	// Reference point
	code.WriteString("    # Reference point\n")
	if mobo.RefPoint.UseHeuristic {
		code.WriteString("    # Heuristic reference point\n")
		code.WriteString("    y_min = Y.min(dim=0).values\n")
		code.WriteString("    y_max = Y.max(dim=0).values\n")
		code.WriteString("    ref_point = (y_min - 0.1 * (y_max - y_min + 1e-6)).to(dtype=torch.double)\n\n")
	} else if mobo.RefPoint.Values != nil {
		code.WriteString("    # Manual reference point\n")
		vals := *mobo.RefPoint.Values
		valStrs := make([]string, len(vals))
		for i, v := range vals {
			valStrs[i] = fmt.Sprintf("%f", v)
		}
		code.WriteString(fmt.Sprintf("    ref_point = torch.tensor([%s], dtype=torch.double)\n\n",
			strings.Join(valStrs, ", ")))
	}

	// MC Sampler
	code.WriteString("    # MC Sampler\n")
	if mobo.MCSampler.Type == "sobol_qmc" {
		code.WriteString(fmt.Sprintf("    sampler = SobolQMCNormalSampler(sample_shape=torch.Size([%d]))\n\n",
			mobo.MCSampler.Samples))
	} else {
		code.WriteString(fmt.Sprintf("    sampler = IIDNormalSampler(sample_shape=torch.Size([%d]))\n\n",
			mobo.MCSampler.Samples))
	}

	// Hypervolume tracking
	code.WriteString("    # Hypervolume tracking\n")
	code.WriteString("    hv_history = []\n")
	code.WriteString("    hv0 = compute_hypervolume(Y, ref_point)\n")
	code.WriteString("    hv_history.append(hv0)\n\n")

	if mobo.Params.Verbose {
		code.WriteString("    print(f'Initial HV: {hv0:.6f}\\n')\n\n")
	}

	// Main optimization loop
	code.WriteString("    # Main MOBO loop\n")
	code.WriteString(fmt.Sprintf("    for it in range(1, %d + 1):\n", mobo.Params.Iterations))

	if mobo.Params.Verbose {
		code.WriteString("        print('-'*80)\n")
		code.WriteString(fmt.Sprintf("        print(f'Iteration {it}/%d')\n", mobo.Params.Iterations))
		code.WriteString("        print('-'*80)\n\n")
	}

	code.WriteString("        t_fit0 = time.time()\n")
	code.WriteString(fmt.Sprintf("        model = initialize_model(X, Y, '%s', '%s', train_Yvar)\n",
		mobo.ModelConfig.Architecture, mobo.ModelConfig.ModelType))
	code.WriteString("        t_fit = time.time() - t_fit0\n\n")

	code.WriteString("        # Set iteration-specific seed\n")
	code.WriteString("        iter_seed = seed + it\n")
	code.WriteString("        torch.manual_seed(iter_seed)\n")
	code.WriteString("        np.random.seed(iter_seed)\n\n")

	// Beta parameter for qUCB
	beta := 0.2
	if mobo.Params.Beta != nil {
		beta = *mobo.Params.Beta
	}

	code.WriteString("        t_acq0 = time.time()\n")
	code.WriteString("        candidates, acq_val = optimize_acquisition(\n")
	code.WriteString("            model=model, X=X, Y=Y, ref_point=ref_point, bounds=bounds,\n")
	code.WriteString(fmt.Sprintf("            acq_type='%s', sampler=sampler, batch_size=%d,\n",
		mobo.Acquisition, mobo.Params.BatchSize))
	code.WriteString(fmt.Sprintf("            num_objectives=M, num_restarts=%d, raw_samples=%d,\n",
		mobo.Params.Restarts, mobo.Params.RawSamples))
	code.WriteString(fmt.Sprintf("            beta=%f, seed=iter_seed\n", beta))
	code.WriteString("        )\n")
	code.WriteString("        t_acq = time.time() - t_acq0\n\n")

	code.WriteString("        # Evaluate new candidates\n")
	code.WriteString("        new_X = candidates.detach()\n")
	code.WriteString("        new_Y = problem(new_X)\n")
	code.WriteString("        X = torch.cat([X, new_X], dim=0)\n")
	code.WriteString("        Y = torch.cat([Y, new_Y], dim=0)\n\n")

	// Update noise variance if using fixed noise
	if mobo.ModelConfig.ModelType == "single_task_fixed_noise" {
		code.WriteString("        # Update noise variance\n")
		code.WriteString("        new_Yvar = torch.full_like(new_Y, train_Yvar[0, 0])\n")
		code.WriteString("        train_Yvar = torch.cat([train_Yvar, new_Yvar], dim=0)\n\n")
	}

	code.WriteString("        # Compute hypervolume\n")
	code.WriteString("        hv = compute_hypervolume(Y, ref_point)\n")
	code.WriteString("        hv_history.append(hv)\n\n")

	if mobo.Params.Verbose {
		code.WriteString("        print(f'Model fit time: {t_fit:.3f}s')\n")
		code.WriteString("        print(f'Acq optimization time: {t_acq:.3f}s')\n")
		code.WriteString("        print(f'Hypervolume: {hv:.6f}')\n")
		code.WriteString("        print(f'Total evaluations: {X.shape[0]}\\n')\n\n")
	}

	// Final results
	code.WriteString("    # Extract Pareto set\n")
	code.WriteString("    pareto_mask = is_non_dominated(Y)\n")
	code.WriteString("    X_pareto = X[pareto_mask]\n")
	code.WriteString("    Y_pareto = Y[pareto_mask]\n\n")

	if mobo.Params.Verbose {
		code.WriteString("    print('='*80)\n")
		code.WriteString("    print('FINAL RESULTS')\n")
		code.WriteString("    print('='*80)\n")
		code.WriteString("    print(f'Total evaluations: {X.shape[0]}')\n")
		code.WriteString("    print(f'Pareto points: {X_pareto.shape[0]}')\n")
		code.WriteString("    print(f'Final HV: {hv_history[-1]:.6f}')\n")
		code.WriteString("    print('='*80 + '\\n')\n\n")
	}

	// Save detailed logbook (verbose only)
	if mobo.Params.Verbose {
		code.WriteString("    # Save detailed logbook\n")
		code.WriteString("    with open(f'{rootPath}/logbook.txt', 'w') as f:\n")
		code.WriteString("        f.write('='*80 + '\\n')\n")
		code.WriteString("        f.write('CONFIGURATION\\n')\n")
		code.WriteString("        f.write('='*80 + '\\n')\n")
		code.WriteString(fmt.Sprintf("        f.write('Problem: %s\\n')\n", mobo.Problem))
		code.WriteString("        f.write(f'Dimensions: {dim}\\n')\n")
		code.WriteString("        f.write(f'Objectives: {M}\\n')\n")
		code.WriteString(fmt.Sprintf("        f.write('Model Architecture: %s\\n')\n", mobo.ModelConfig.Architecture))
		code.WriteString(fmt.Sprintf("        f.write('Model Type: %s\\n')\n", mobo.ModelConfig.ModelType))
		code.WriteString(fmt.Sprintf("        f.write('Acquisition: %s\\n')\n", mobo.Acquisition))
		code.WriteString(fmt.Sprintf("        f.write('Initial Samples: %d (%s)\\n')\n", mobo.Params.InitialPoints, strings.ToUpper(mobo.InitialDesign.Strategy)))
		code.WriteString(fmt.Sprintf("        f.write('Iterations: %d\\n')\n", mobo.Params.Iterations))
		code.WriteString(fmt.Sprintf("        f.write('Batch Size: %d\\n')\n", mobo.Params.BatchSize))
		code.WriteString("        f.write('='*80 + '\\n\\n')\n")
		code.WriteString("        \n")
		code.WriteString("        f.write('='*80 + '\\n')\n")
		code.WriteString("        f.write('ITERATION LOG\\n')\n")
		code.WriteString("        f.write('='*80 + '\\n')\n")
		code.WriteString("        # Header\n")
		code.WriteString("        f.write(f\"{'Iter':<6} {'Input Point':<40} \")\n")
		code.WriteString("        for obj_idx in range(M):\n")
		code.WriteString("            f.write(f\"{'Obj'+str(obj_idx+1):<12} \")\n")
		code.WriteString("        f.write(f\"{'HV':<12} {'Type':<10}\\n\")\n")
		code.WriteString("        f.write('-'*80 + '\\n')\n")
		code.WriteString("        \n")
		code.WriteString("        # Log each iteration\n")
		code.WriteString("        X_np = X.detach().cpu().numpy()\n")
		code.WriteString("        Y_np = Y.detach().cpu().numpy()\n")
		code.WriteString("        for i in range(len(X_np)):\n")
		code.WriteString("            point_str = '[' + ', '.join([f'{v:7.3f}' for v in X_np[i]]) + ']'\n")
		code.WriteString(fmt.Sprintf("            iter_type = 'Initial' if i < %d else 'MOBO'\n", mobo.Params.InitialPoints))
		code.WriteString("            f.write(f\"{i+1:<6} {point_str:<40} \")\n")
		code.WriteString("            for obj_idx in range(M):\n")
		code.WriteString("                f.write(f\"{Y_np[i][obj_idx]:<12.4f} \")\n")
		code.WriteString("            # Compute HV up to this point\n")
		code.WriteString("            hv_curr = compute_hypervolume(Y[:i+1], ref_point) if i > 0 else hv_history[0]\n")
		code.WriteString("            f.write(f\"{hv_curr:<12.6f} {iter_type:<10}\\n\")\n")
		code.WriteString("        \n")
		code.WriteString("        f.write('='*80 + '\\n\\n')\n")
		code.WriteString("        \n")
		code.WriteString("        f.write('='*80 + '\\n')\n")
		code.WriteString("        f.write('OPTIMIZATION SUMMARY\\n')\n")
		code.WriteString("        f.write('='*80 + '\\n')\n")
		code.WriteString("        f.write(f'Total Evaluations: {X.shape[0]}\\n')\n")
		code.WriteString("        f.write(f'Pareto Points: {X_pareto.shape[0]}\\n')\n")
		code.WriteString("        f.write(f'Final Hypervolume: {hv_history[-1]:.6f}\\n')\n")
		code.WriteString("        f.write(f'Initial Hypervolume: {hv_history[0]:.6f}\\n')\n")
		code.WriteString("        f.write(f'HV Improvement: {hv_history[-1] - hv_history[0]:.6f}\\n')\n")
		code.WriteString("        f.write('-'*80 + '\\n')\n")
		code.WriteString("        f.write('PARETO FRONT SOLUTIONS:\\n')\n")
		code.WriteString("        f.write('-'*80 + '\\n')\n")
		code.WriteString("        X_pareto_np = X_pareto.detach().cpu().numpy()\n")
		code.WriteString("        Y_pareto_np = Y_pareto.detach().cpu().numpy()\n")
		code.WriteString("        for i in range(len(X_pareto_np)):\n")
		code.WriteString("            point_str = '[' + ', '.join([f'{v:7.3f}' for v in X_pareto_np[i]]) + ']'\n")
		code.WriteString("            f.write(f\"  {i+1}. x={point_str}\\n\")\n")
		code.WriteString("            obj_str = '     f(x)=[' + ', '.join([f'{v:8.4f}' for v in Y_pareto_np[i]]) + ']'\n")
		code.WriteString("            f.write(f\"{obj_str}\\n\")\n")
		code.WriteString("        f.write('='*80 + '\\n')\n\n")
	}

	// Save best solutions summary (always saved, like SBO's best.txt)
	code.WriteString("    # Save best solutions summary\n")
	code.WriteString("    with open(f'{rootPath}/best.txt', 'w') as f:\n")
	code.WriteString("        f.write('PARETO OPTIMAL SOLUTIONS\\n')\n")
	code.WriteString("        f.write('='*80 + '\\n\\n')\n")
	code.WriteString("        f.write(f'Total Evaluations: {X.shape[0]}\\n')\n")
	code.WriteString("        f.write(f'Pareto Points Found: {X_pareto.shape[0]}\\n')\n")
	code.WriteString("        f.write(f'Final Hypervolume: {hv_history[-1]:.6f}\\n\\n')\n")
	code.WriteString("        f.write('Pareto Front Solutions:\\n')\n")
	code.WriteString("        f.write('-'*80 + '\\n')\n")
	code.WriteString("        X_pareto_np = X_pareto.detach().cpu().numpy()\n")
	code.WriteString("        Y_pareto_np = Y_pareto.detach().cpu().numpy()\n")
	code.WriteString("        for i in range(len(X_pareto_np)):\n")
	code.WriteString("            f.write(f\"\\nSolution {i+1}:\\n\")\n")
	code.WriteString("            f.write(f\"  Input: {X_pareto_np[i].tolist()}\\n\")\n")
	code.WriteString("            f.write(f\"  Objectives: {Y_pareto_np[i].tolist()}\\n\")\n\n")

	// Save Pareto front plot (if 2D or 3D)
	code.WriteString("    # Save Pareto front plot\n")
	code.WriteString("    if M == 2:\n")
	code.WriteString(fmt.Sprintf("        plot_pareto_front_2d(Y, ref_point, f'{rootPath}/pareto_front.png', %d)\n",
		mobo.Params.Iterations))
	code.WriteString("    elif M == 3:\n")
	code.WriteString(fmt.Sprintf("        plot_pareto_front_3d(Y, ref_point, f'{rootPath}/pareto_front.png', %d)\n\n",
		mobo.Params.Iterations))

	// Save final HV convergence plot
	code.WriteString("    # Save final hypervolume convergence plot\n")
	code.WriteString("    plot_hypervolume(hv_history, f'{rootPath}/hypervolume_convergence.png')\n\n")

	if mobo.Params.Verbose {
		code.WriteString("    print('✅ Results saved to:')\n")
		code.WriteString("    print(f'  - {rootPath}/best.txt')\n")
		code.WriteString("    print(f'  - {rootPath}/logbook.txt')\n")
		code.WriteString("    print(f'  - {rootPath}/hypervolume_convergence.png')\n")
		code.WriteString("    if M in [2, 3]:\n")
		code.WriteString("        print(f'  - {rootPath}/pareto_front.png')\n")
		code.WriteString("    print('\\n✅ Done.\\n')\n\n")
	}

	return code.String()
}

func (mobo *MOBO) Code() (string, error) {
	if err := mobo.validate(); err != nil {
		return "", err
	}

	var code strings.Builder

	code.WriteString(mobo.imports())
	code.WriteString(mobo.customFunctionCode())
	code.WriteString(mobo.helperFunctions())
	code.WriteString(mobo.generateInitialData())
	code.WriteString(mobo.initializeModel())
	code.WriteString(mobo.acquisitionFunction())
	code.WriteString(mobo.mainFunction())
	code.WriteString("if __name__ == '__main__':\n")
	code.WriteString("    import sys\n")
	code.WriteString("    import os\n")
	code.WriteString("    exit_code = 0\n")
	code.WriteString("    try:\n")
	code.WriteString("        main()\n")
	code.WriteString("        print('\\n✅ Optimization completed successfully!')\n")
	code.WriteString("    except Exception as e:\n")
	code.WriteString("        import traceback\n")
	code.WriteString("        \n")
	code.WriteString("        # Capture error details safely\n")
	code.WriteString("        try:\n")
	code.WriteString("            error_type = type(e).__name__\n")
	code.WriteString("            error_msg = str(e)\n")
	code.WriteString("            error_trace = traceback.format_exc()\n")
	code.WriteString("            \n")
	code.WriteString("            error_details = 'MOBO Error Details' + '\\n'\n")
	code.WriteString("            error_details += '=' * 80 + '\\n'\n")
	code.WriteString("            error_details += 'Exception Type: ' + error_type + '\\n'\n")
	code.WriteString("            error_details += 'Exception Message: ' + error_msg + '\\n'\n")
	code.WriteString("            error_details += '=' * 80 + '\\n' + '\\n'\n")
	code.WriteString("            error_details += 'Full Traceback:' + '\\n'\n")
	code.WriteString("            error_details += error_trace\n")
	code.WriteString("        except Exception as format_error:\n")
	code.WriteString("            error_details = 'Error formatting exception: ' + str(format_error) + '\\n'\n")
	code.WriteString("            error_details += 'Original error: ' + str(e) + '\\n'\n")
	code.WriteString("        \n")
	code.WriteString("        # Write to error.log\n")
	code.WriteString("        try:\n")
	code.WriteString("            with open('error.log', 'w') as f:\n")
	code.WriteString("                f.write(error_details)\n")
	code.WriteString("                f.flush()\n")
	code.WriteString("        except Exception as write_error:\n")
	code.WriteString("            print('Failed to write error.log: ' + str(write_error), file=sys.stderr)\n")
	code.WriteString("        \n")
	code.WriteString("        # Print to stderr\n")
	code.WriteString("        print('\\n⚠️  ERROR OCCURRED:', file=sys.stderr)\n")
	code.WriteString("        print(error_details, file=sys.stderr)\n")
	code.WriteString("        sys.stderr.flush()\n")
	code.WriteString("        \n")
	code.WriteString("        # Check if critical results exist\n")
	code.WriteString("        try:\n")
	code.WriteString("            results_saved = os.path.exists('best.txt') and os.path.exists('logbook.txt') and os.path.getsize('best.txt') > 0\n")
	code.WriteString("        except:\n")
	code.WriteString("            results_saved = False\n")
	code.WriteString("        \n")
	code.WriteString("        if results_saved:\n")
	code.WriteString("            print('\\n✅ Core optimization results saved successfully.', file=sys.stderr)\n")
	code.WriteString("            print('Note: Error occurred in post-processing phase.', file=sys.stderr)\n")
	code.WriteString("            exit_code = 0\n")
	code.WriteString("        else:\n")
	code.WriteString("            print('\\n❌ Optimization failed - no results generated.', file=sys.stderr)\n")
	code.WriteString("            exit_code = 1\n")
	code.WriteString("    \n")
	code.WriteString("    sys.exit(exit_code)\n")

	return code.String(), nil
}
