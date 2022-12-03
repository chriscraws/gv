package example

type Float = float64
type Vec2 [2]Float
type Vec3 [3]Float
type Vec4 [4]Float
type Mat2 [4]Float
type Mat3 [9]Float
type Mat4 [16]Float

type Int = int32
type IVec2 [2]Int
type IVec3 [3]Int
type IVec4 [4]Int
type IMat2 [4]Int
type IMat3 [9]Int
type IMat4 [16]Int

type UInt = uint32
type UIVec2 [2]UInt
type UIVec3 [3]UInt
type UIVec4 [4]UInt
type UIMat2 [4]UInt
type UIMat3 [9]UInt
type UIMat4 [16]UInt

type FragmentIO struct {
	Output    []Vec4
	Depth     Float
	FragCoord Vec2
}

type ComputeIO struct {
	LocalID     UIVec3
	GroupID     UIVec3
	GlobalID    UIVec3
	GlobalIndex UInt
}

type PhongModel struct {
	Ambient  Vec3
	Diffuse  Vec3
	Specular Vec3
}

// Decorations

type Uniform struct{}

type Buffer[T any] struct {
	Data *T
}

type BindBuffer[T any] struct {
	Target *Buffer[T]
}

type ExampleModule struct {
	ComputeIO
	Phong struct {
		Uniform
		BindBuffer[PhongModel]
	}
}
