package expr

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"reflect"
	"strings"

	"fmt"

	"github.com/TIBCOSoftware/flogo-lib/core/mapper/exprmapper/expression/function"
	"github.com/TIBCOSoftware/flogo-lib/core/mapper/exprmapper/util"
	"github.com/TIBCOSoftware/flogo-lib/core/mapper/exprmapper/ref"
	"github.com/TIBCOSoftware/flogo-lib/core/data"
	"github.com/TIBCOSoftware/flogo-lib/logger"
)

var log = logger.GetLogger("expr")

type OPERATIOR int

const (
	EQ OPERATIOR = iota
	OR
	AND
	NOT_EQ
	GT
	LT
	GTE
	LTE
	ADDITION
	SUBTRACTION
	MULTIPLICATION
	DIVISION
	INT_DIVISTION
	MODULAR_DIVISION
	GEGATIVE
	UNINE
)

var operatorMap = map[string]OPERATIOR{
	"eq":   EQ,
	"or":   OR,
	"and":  AND,
	"ne":   NOT_EQ,
	"gt":   GT,
	"lt":   LT,
	"ge":   GTE,
	"le":   LTE,
	"+":    ADDITION,
	"-":    SUBTRACTION,
	"*":    MULTIPLICATION,
	"div":  DIVISION,
	"idiv": INT_DIVISTION,
	"mod":  MODULAR_DIVISION,
	"|":    UNINE,
	//TODO negtive
}

var operatorCharactorMap = map[string]OPERATIOR{
	"==":  EQ,
	"=":   EQ,
	"||":  OR,
	"&":   AND,
	"!=":  NOT_EQ,
	">":   GT,
	"<":   LT,
	">=":  GTE,
	"<=":  LTE,
	"+":   ADDITION,
	"-":   SUBTRACTION,
	"*":   MULTIPLICATION,
	"/":   DIVISION,
	"//":  INT_DIVISTION,
	"mod": MODULAR_DIVISION,
	"|":   UNINE,
	//TODO negtive
}

func ToOperator(operator string) (OPERATIOR, bool) {
	op, found := operatorMap[operator]
	if !found {
		op, found = operatorCharactorMap[operator]
	}
	return op, found
}

func (o OPERATIOR) String() string {
	for k, v := range operatorCharactorMap {
		if v == o {
			return k
		}
	}
	return ""
}

type Expression struct {
	Left     *Expression `json:"left"`
	Operator OPERATIOR   `json:"operator"`
	Right    *Expression `json:"right"`

	Value interface{} `json:"value"`
	Type  data.Type `json:"type"`

	//done
}

func (e *Expression) IsNil() bool {
	if e.Left == nil && e.Right == nil {
		return true
	}
	return false
}

type TernaryExpressio struct {
	First  interface{}
	Second interface{}
	Third  interface{}
}

func (t *TernaryExpressio) EvalWithScope(inputScope data.Scope, resolver data.Resolver) (interface{}, error) {
	v, err := t.HandleParameter(t.First, inputScope, resolver)
	if err != nil {
		return nil, err
	}
	if v.(bool) {
		v2, err2 := t.HandleParameter(t.Second, inputScope, resolver)
		if err2 != nil {
			return nil, err2
		}
		return v2, nil
	} else {
		v3, err3 := t.HandleParameter(t.Third, inputScope, resolver)
		if err3 != nil {
			return nil, err3
		}
		return v3, nil
	}
}

func (t *TernaryExpressio) HandleParameter(param interface{}, inputScope data.Scope, resolver data.Resolver) (interface{}, error) {
	var firstValue interface{}
	fmt.Println(reflect.TypeOf(param))
	switch t := param.(type) {
	case *function.FunctionExp:
		vss, err := t.EvalWithScope(inputScope, resolver)
		if err != nil {
			return nil, err
		}
		if len(vss) > 0 {
			firstValue = vss[0]
		}
		return firstValue, nil
	case *Expression:
		vss, err := t.EvalWithScope(inputScope, resolver)
		if err != nil {
			return nil, err
		}
		firstValue = vss
		return firstValue, nil

	default:
		firstValue = t
		return firstValue, nil
	}
}

func (e *Expression) Serialization() (string, error) {
	v, err := json.Marshal(e)
	return base64.StdEncoding.EncodeToString(v), err
}

func (e *Expression) String() string {
	v, err := json.Marshal(e)
	if err != nil {
		log.Errorf("Expression to string error [%s]", err.Error())
		return ""
	}
	return string(v)
}

func DeSerialization(base64str string) (*Expression, error) {
	ex := &Expression{}

	v, err := base64.StdEncoding.DecodeString(base64str)
	if err != nil {
		return nil, errors.New("Do serialization function err: " + err.Error())
	}
	err = json.Unmarshal(v, ex)
	if err != nil {
		return nil, errors.New("Do Unmarshal function err: " + err.Error())
	}
	return ex, nil
}

func (e *Expression) UnmarshalJSON(exprData []byte) error {
	ser := &struct {
		Left     *Expression `json:"left"`
		Operator OPERATIOR   `json:"operator"`
		Right    *Expression `json:"right"`
		Value    interface{} `json:"value"`
		Type     data.Type `json:"type"`
	}{}

	if err := json.Unmarshal(exprData, ser); err != nil {
		return err
	}

	e.Left = ser.Left
	e.Right = ser.Right
	e.Operator = ser.Operator

	v, err := function.ConvertToValue(ser.Value, ser.Type)
	if err != nil {
		return err
	}
	e.Value = v
	e.Type = ser.Type

	return nil
}

func NewWIExpression() *Expression {
	return &Expression{}
}

func (e *Expression) IsFunction() bool {
	if data.FUNCTION == e.Type {
		return true
	}
	return false
}

func (f *Expression) Eval() (interface{}, error) {
	log.Debug("Expression eval method....")
	return f.evaluate(nil, nil, nil)
}

func (f *Expression) EvalWithScope(inputScope data.Scope, resolver data.Resolver) (interface{}, error) {
	log.Debug("Expression eval method....")
	return f.evaluate(nil, inputScope, resolver)
}

func (f *Expression) EvalWithData(data interface{}, inputScope data.Scope, resolver data.Resolver) (interface{}, error) {
	log.Debug("Expression eval method....")
	return f.evaluate(data, inputScope, resolver)
}

func (f *Expression) evaluate(data interface{}, inputScope data.Scope, resolver data.Resolver) (interface{}, error) {
	log.Debug("Expression evaluate method....")
	//Left
	leftResultChan := make(chan interface{}, 1)
	rightResultChan := make(chan interface{}, 1)
	if f.IsNil() {
		log.Debugf("Expression right and left are nil, return value directly")
		return f.Value, nil
	}
	go f.Left.do(data, inputScope, resolver, leftResultChan)
	go f.Right.do(data, inputScope, resolver, rightResultChan)

	leftValue := <-leftResultChan
	rightValue := <-rightResultChan

	//Make sure no error returned
	switch leftValue.(type) {
	case error:
		return nil, leftValue.(error)
	}

	switch rightValue.(type) {
	case error:
		return nil, rightValue.(error)
	}

	log.Debugf("Left value ", leftValue, " Type ", reflect.TypeOf(leftValue))
	log.Debugf("Right value ", rightValue, " Type ", reflect.TypeOf(rightValue))
	//Operator
	operator := f.Operator

	return f.run(leftValue, operator, rightValue)
}

func (f *Expression) do(edata interface{}, inputScope data.Scope, resolver data.Resolver, resultChan chan interface{}) {
	if f == nil {
		resultChan <- nil
	}
	log.Debug("Do left and expression ", f)
	var leftValue interface{}
	if f.IsFunction() {
		function := f.Value.(*function.FunctionExp)
		funcReturn, err := function.EvalWithScope(inputScope, resolver)
		if err != nil {
			resultChan <- errors.New("Eval left expression error: " + err.Error())
		}

		if len(funcReturn) > 1 {
			resultChan <- errors.New("Function " + function.Name + " cannot return more than one using in expression")
		}
		if len(funcReturn) == 1 {
			leftValue = funcReturn[0]
		}
	} else if f.Type == data.EXPRESSION {
		var err error
		leftValue, err = f.evaluate(edata, inputScope, resolver)
		if err != nil {
			resultChan <- errors.New("Eval left expression error: " + err.Error())
		}
	} else if f.Type == data.REF {
		refMaping := ref.NewMappingRef(f.Value.(string))
		v, err := refMaping.Eval(inputScope, resolver)
		if err != nil {
			log.Errorf("Mapping ref eva error [%s]", err.Error())
			resultChan <- fmt.Errorf("Mapping ref eva error [%s]", err.Error())
		}
		leftValue = v
	} else if f.Type == data.ARRAYREF {
		arrayRef := ref.NewArrayRef(f.Value.(string))
		v, err := arrayRef.EvalFromData(edata)
		if err != nil {
			log.Errorf("Mapping ref eva error [%s]", err.Error())
			resultChan <- fmt.Errorf("Mapping ref eva error [%s]", err.Error())
		}
		leftValue = v
	} else {
		leftValue = f.Value
	}

	resultChan <- leftValue
}

func (f *Expression) run(left interface{}, op OPERATIOR, right interface{}) (interface{}, error) {
	switch op {
	case EQ:
		return equals(left, right)
	case OR:
		return or(left, right)
	case AND:
		return add(left, right)
	case NOT_EQ:
		return notEquals(left, right)
	case GT:
		return gt(left, right, false)
	case LT:
		return lt(left, right, false)
	case GTE:
		return gt(left, right, true)
	case LTE:
		return lt(left, right, true)
	case ADDITION:
		return additon(left, right)
	case SUBTRACTION:
		return sub(left, right)
	case MULTIPLICATION:
		return multiplication(left, right)
	case DIVISION:
		return div(left, right)
	case INT_DIVISTION:
		//TODO....
		return add(left, right)
	case MODULAR_DIVISION:
		//TODO....
		return add(left, right)
	case GEGATIVE:
		//TODO....
		return add(left, right)
	case UNINE:
		//TODO....
		return add(left, right)
	default:
		return nil, errors.New("Unknow operator " + op.String())
	}

	return nil, nil

}

func equals(left interface{}, right interface{}) (bool, error) {
	log.Debugf("Left expression value %+v, right expression value %+v", left, right)
	if left == nil && right == nil {
		return true, nil
	} else if left == nil && right != nil {
		return false, nil
	} else if left != nil && right == nil {
		return false, nil
	}

	rightType := getType(right)
	log.Debugf("Right expression type [%s]", rightType)
	switch left.(type) {
	case int:
		if rightType.Kind() == reflect.Int {
			return left.(int) == right.(int), nil
		}
	case int64:

		if rightType.Kind() == reflect.Int64 {
			return left.(int64) == right.(int64), nil
		}
	case float64:
		if rightType.Kind() == reflect.Float64 {
			return left.(float64) == right.(float64), nil
		}
	case string:
		s, err := util.ConvertToString(right)
		if err != nil {
			return false, err
		}
		return strings.EqualFold(left.(string), s), nil
	case bool:
		if rightType.Kind() == reflect.Bool {
			return left.(bool) == right.(bool), nil
		}
	default:
		return false, errors.New("Unknow type to equals" + getType(left).String())
	}

	return false, nil
}

func notEquals(left interface{}, right interface{}) (bool, error) {

	log.Debugf("Left expression value %+v, right expression value %+v", left, right)
	if left == nil && right == nil {
		return false, nil
	} else if left == nil && right != nil {
		return true, nil
	} else if left != nil && right == nil {
		return true, nil
	}

	rightType := getType(right)
	switch left.(type) {
	case int:
		if rightType.Kind() == reflect.Int {
			return left.(int) != right.(int), nil
		} else {
			return false, errors.New("Right expression must be int")
		}
	case int64:

		if rightType.Kind() != reflect.Int64 {
			return left.(int64) == right.(int64), nil
		} else {
			return false, errors.New("Right expression must be int64")
		}
	case float64:
		if rightType.Kind() == reflect.Float64 {
			return left.(float64) != right.(float64), nil
		} else {
			return false, errors.New("Right expression must be float64")
		}
	case string:
		if rightType.Kind() == reflect.String {
			return !strings.EqualFold(left.(string), right.(string)), nil
		} else {
			return false, errors.New("Right expression must be string")
		}
	case bool:
		if rightType.Kind() == reflect.Bool {
			return left.(bool) != right.(bool), nil
		} else {
			return false, errors.New("Right expression must be int")
		}
	default:
		return false, errors.New("Unknow type to equals" + getType(left).String())
	}

	return false, nil
}

func gt(left interface{}, right interface{}, includeEquals bool) (bool, error) {

	log.Debugf("Left expression value %+v, right expression value %+v", left, right)
	if left == nil && right == nil {
		return false, nil
	} else if left == nil && right != nil {
		return false, nil
	} else if left != nil && right == nil {
		return false, nil
	}

	rightType := getType(right)
	switch left.(type) {
	case int:
		if rightType.Kind() == reflect.Int {
			if includeEquals {
				return left.(int) >= right.(int), nil

			} else {
				return left.(int) > right.(int), nil
			}
		} else {
			return false, errors.New("Right expression must be int")
		}
	case int64:
		if rightType.Kind() == reflect.Int64 {
			if includeEquals {
				return left.(int64) >= right.(int64), nil

			} else {
				return left.(int64) > right.(int64), nil
			}
		} else {
			return false, errors.New("Right expression must be int64")
		}
	case float64:
		if rightType.Kind() == reflect.Float64 {
			if includeEquals {
				return left.(float64) >= right.(float64), nil

			} else {
				return left.(float64) > right.(float64), nil
			}
		} else {
			return false, errors.New("Right expression must be float64")
		}
	default:
		return false, errors.New("Unknow type to equals" + getType(left).String())
	}

	return false, nil
}

func lt(left interface{}, right interface{}, includeEquals bool) (bool, error) {

	log.Debugf("Left expression value %+v, right expression value %+v", left, right)
	if left == nil && right == nil {
		return false, nil
	} else if left == nil && right != nil {
		return false, nil
	} else if left != nil && right == nil {
		return false, nil
	}

	rightType := getType(right)
	switch left.(type) {
	case int:
		if rightType.Kind() == reflect.Int {
			if includeEquals {
				return left.(int) <= right.(int), nil

			} else {
				return left.(int) < right.(int), nil
			}
		} else {
			return false, errors.New("Right expression must be int")
		}
	case int64:
		if rightType.Kind() == reflect.Int64 {
			if includeEquals {
				return left.(int64) <= right.(int64), nil

			} else {
				return left.(int64) < right.(int64), nil
			}
		} else {
			return false, errors.New("Right expression must be int64")
		}
	case float64:
		if rightType.Kind() == reflect.Float64 {
			if includeEquals {
				return left.(float64) <= right.(float64), nil

			} else {
				return left.(float64) < right.(float64), nil
			}
		} else {
			return false, errors.New("Right expression must be float64")
		}
	default:
		return false, errors.New("Unknow type to equals" + getType(left).String())
	}

	return false, nil
}

func add(left interface{}, right interface{}) (bool, error) {

	log.Infof("Add operator, Left expression value %+v, right expression value %+v", left, right)

	rightType := getType(right)
	switch left.(type) {
	case bool:
		if rightType.Kind() == reflect.Bool {
			return left.(bool) && right.(bool), nil
		}
	default:
		return false, errors.New("Unknow type to add expression " + getType(left).String())
	}

	return false, nil
}

func or(left interface{}, right interface{}) (bool, error) {

	log.Infof("Add operator, Left expression value %+v, right expression value %+v", left, right)

	rightType := getType(right)
	switch left.(type) {
	case bool:
		if rightType.Kind() == reflect.Bool {
			return left.(bool) || right.(bool), nil
		} else {
			return false, errors.New("Unknow type to or expression " + getType(rightType).String())
		}
	default:
		return false, errors.New("Unknow type to add expression " + getType(left).String())
	}

	return false, nil
}

func additon(left interface{}, right interface{}) (interface{}, error) {

	log.Infof("Left expression value %+v, right expression value %+v", left, right)
	if left == nil && right == nil {
		return false, nil
	} else if left == nil && right != nil {
		return false, nil
	} else if left != nil && right == nil {
		return false, nil
	}

	rightType := getType(right)
	switch left.(type) {
	case int:
		if rightType.Kind() == reflect.Int {
			return left.(int) + right.(int), nil

		} else {
			return false, errors.New("Right expression must be int")
		}
	case int64:
		if rightType.Kind() == reflect.Int64 {
			return left.(int64) + right.(int64), nil
		} else {
			return false, errors.New("Right expression must be int64")
		}
	case float64:
		if rightType.Kind() == reflect.Float64 {
			return left.(float64) + right.(float64), nil
		} else {
			return false, errors.New("Right expression must be float64")
		}
	default:
		return false, errors.New("Unknow type to equals" + getType(left).String())
	}

	return false, nil
}

func sub(left interface{}, right interface{}) (interface{}, error) {

	log.Debugf("Left expression value %+v, right expression value %+v", left, right)
	if left == nil && right == nil {
		return false, nil
	} else if left == nil && right != nil {
		return false, nil
	} else if left != nil && right == nil {
		return false, nil
	}

	rightType := getType(right)
	switch left.(type) {
	case int:
		if rightType.Kind() == reflect.Int {
			return left.(int) - right.(int), nil

		} else {
			return false, errors.New("Right expression must be int")
		}
	case int64:
		if rightType.Kind() == reflect.Int64 {
			return left.(int64) - right.(int64), nil
		} else {
			return false, errors.New("Right expression must be int64")
		}
	case float64:
		if rightType.Kind() == reflect.Float64 {
			return left.(float64) - right.(float64), nil
		} else {
			return false, errors.New("Right expression must be float64")
		}
	default:
		return false, errors.New("Unknow type to equals" + getType(left).String())
	}

	return false, nil
}

func multiplication(left interface{}, right interface{}) (interface{}, error) {

	log.Infof("Left expression value %+v, right expression value %+v", left, right)
	if left == nil && right == nil {
		return false, nil
	} else if left == nil && right != nil {
		return false, nil
	} else if left != nil && right == nil {
		return false, nil
	}

	rightType := getType(right)
	switch left.(type) {
	case int:
		if rightType.Kind() == reflect.Int {
			return left.(int) * right.(int), nil

		} else {
			return false, errors.New("Right expression must be int")
		}
	case int64:
		if rightType.Kind() == reflect.Int64 {
			return left.(int64) * right.(int64), nil
		} else {
			return false, errors.New("Right expression must be int64")
		}
	case float64:
		if rightType.Kind() == reflect.Float64 {
			return left.(float64) * right.(float64), nil
		} else {
			return false, errors.New("Right expression must be float64")
		}
	default:
		return false, errors.New("Unknow type to equals" + getType(left).String())
	}

	return false, nil
}

func div(left interface{}, right interface{}) (interface{}, error) {

	log.Debugf("Left expression value %+v, right expression value %+v", left, right)
	if left == nil && right == nil {
		return false, nil
	} else if left == nil && right != nil {
		return false, nil
	} else if left != nil && right == nil {
		return false, nil
	}

	rightType := getType(right)
	switch left.(type) {
	case int:
		if rightType.Kind() == reflect.Int {
			return left.(int) + right.(int), nil

		} else {
			return false, errors.New("Right expression must be int")
		}
	case int64:
		if rightType.Kind() == reflect.Int64 {
			return left.(int64) + right.(int64), nil
		} else {
			return false, errors.New("Right expression must be int64")
		}
	case float64:
		if rightType.Kind() == reflect.Float64 {
			return left.(float64) + right.(float64), nil
		} else {
			return false, errors.New("Right expression must be float64")
		}
	default:
		return false, errors.New("Unknow type to equals" + getType(left).String())
	}

	return false, nil
}

func getType(in interface{}) reflect.Type {
	return reflect.TypeOf(in)
}